// Package msg provides Matrix messaging commands for jack.
package msg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Package-level config set by the parent package during PersistentPreRunE.
var (
	Homeserver        string
	RegistrationToken string
)

// Cmd is the parent command for all messaging subcommands.
var Cmd = &cobra.Command{
	Use:   "msg",
	Short: "Matrix messaging commands",
	Long:  "Manage Matrix messaging: register users, send and read messages, create rooms.",
}

// Client performs HTTP requests against a Matrix homeserver.
type Client struct {
	HTTP        *http.Client
	Homeserver  string
	AccessToken string
}

// NewClient creates a Matrix client for the given homeserver and access token.
func NewClient(homeserver, accessToken string) *Client {
	return &Client{
		Homeserver:  strings.TrimRight(homeserver, "/"),
		AccessToken: accessToken,
		HTTP:        http.DefaultClient,
	}
}

// --- Response types ---

// Registration is returned by register and login endpoints.
type Registration struct {
	UserID      string `json:"user_id"`
	AccessToken string `json:"access_token"`
	DeviceID    string `json:"device_id"`
}

// Room is returned when creating a room.
type Room struct {
	RoomID string `json:"room_id"`
}

// JoinedRooms is returned when listing joined rooms.
type JoinedRooms struct {
	Rooms []string `json:"joined_rooms"`
}

// RoomInfo holds a room's name and topic.
type RoomInfo struct {
	Name  string `json:"name"`
	Topic string `json:"topic"`
}

// Member represents a room member with their display name.
type Member struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

// Message represents a single Matrix timeline event.
type Message struct {
	Sender  string                 `json:"sender"`
	Type    string                 `json:"type"`
	Content map[string]interface{} `json:"content"`
	EventID string                 `json:"event_id"`
}

// Messages is the response from the room messages endpoint.
type Messages struct {
	End   string    `json:"end"`
	Chunk []Message `json:"chunk"`
}

// --- Dependency types for testability ---

// Registerer registers a Matrix user account.
type Registerer func(username, password, token string) (*Registration, error)

// Authenticator logs into a Matrix account.
type Authenticator func(username, password string) (*Registration, error)

// RoomCreator creates a Matrix room.
type RoomCreator func(name, topic string) (*Room, error)

// Inviter invites a user to a Matrix room.
type Inviter func(roomID, userID string) error

// MessageSender sends a message to a Matrix room.
type MessageSender func(roomID, message string) (string, error)

// MessageReader reads messages from a Matrix room.
type MessageReader func(roomID string, limit int) (*Messages, error)

// RoomLister lists joined Matrix rooms.
type RoomLister func() (*JoinedRooms, error)

// RoomInfoGetter retrieves room name and topic.
type RoomInfoGetter func(roomID string) (*RoomInfo, error)

// MemberLister lists the members of a room.
type MemberLister func(roomID string) ([]Member, error)


// --- Client methods ---

// Register creates a new Matrix user account.
func (c *Client) Register(username, password, token string) (*Registration, error) {
	body := map[string]interface{}{
		"auth": map[string]interface{}{
			"type":               "m.login.registration_token",
			"registration_token": token,
		},
		"username": username,
		"password": password,
	}
	var reg Registration
	if err := c.post("/_matrix/client/v3/register", body, &reg); err != nil {
		return nil, fmt.Errorf("register: %w", err)
	}
	return &reg, nil
}

// Login authenticates and returns an access token.
func (c *Client) Login(username, password string) (*Registration, error) {
	body := map[string]interface{}{
		"type":     "m.login.password",
		"user":     username,
		"password": password,
	}
	var reg Registration
	if err := c.post("/_matrix/client/v3/login", body, &reg); err != nil {
		return nil, fmt.Errorf("login: %w", err)
	}
	return &reg, nil
}

// CreateRoom creates a new Matrix room with an optional topic.
func (c *Client) CreateRoom(name, topic string) (*Room, error) {
	body := map[string]interface{}{
		"name":   name,
		"preset": "private_chat",
	}
	if topic != "" {
		body["topic"] = topic
	}
	var room Room
	if err := c.post("/_matrix/client/v3/createRoom", body, &room); err != nil {
		return nil, fmt.Errorf("create room: %w", err)
	}
	return &room, nil
}

// Invite invites a user to a room.
func (c *Client) Invite(roomID, userID string) error {
	body := map[string]interface{}{
		"user_id": userID,
	}
	if err := c.post(fmt.Sprintf("/_matrix/client/v3/rooms/%s/invite", url.PathEscape(roomID)), body, nil); err != nil {
		return fmt.Errorf("invite: %w", err)
	}
	return nil
}

// Send sends a text message to a room and returns the event ID.
func (c *Client) Send(roomID, message string) (string, error) {
	txnID := fmt.Sprintf("%d", time.Now().UnixNano())
	body := map[string]interface{}{
		"msgtype": "m.text",
		"body":    message,
	}
	var resp struct {
		EventID string `json:"event_id"`
	}
	path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/send/m.room.message/%s", url.PathEscape(roomID), txnID)
	if err := c.put(path, body, &resp); err != nil {
		return "", fmt.Errorf("send: %w", err)
	}
	return resp.EventID, nil
}

// Messages retrieves recent messages from a room.
func (c *Client) Messages(roomID string, limit int) (*Messages, error) {
	path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/messages?dir=b&limit=%d", url.PathEscape(roomID), limit)
	var msgs Messages
	if err := c.get(path, &msgs); err != nil {
		return nil, fmt.Errorf("messages: %w", err)
	}
	return &msgs, nil
}

// JoinedRooms returns the list of rooms the user has joined.
func (c *Client) JoinedRooms() (*JoinedRooms, error) {
	var rooms JoinedRooms
	if err := c.get("/_matrix/client/v3/joined_rooms", &rooms); err != nil {
		return nil, fmt.Errorf("joined rooms: %w", err)
	}
	return &rooms, nil
}

// GetRoomInfo retrieves the name and topic for a room.
func (c *Client) GetRoomInfo(roomID string) (*RoomInfo, error) {
	var info RoomInfo

	// Fetch room name (ignore errors — name may not be set).
	var nameEvent struct {
		Name string `json:"name"`
	}
	namePath := fmt.Sprintf("/_matrix/client/v3/rooms/%s/state/m.room.name", url.PathEscape(roomID))
	if err := c.get(namePath, &nameEvent); err == nil {
		info.Name = nameEvent.Name
	}

	// Fetch room topic (ignore errors — topic may not be set).
	var topicEvent struct {
		Topic string `json:"topic"`
	}
	topicPath := fmt.Sprintf("/_matrix/client/v3/rooms/%s/state/m.room.topic", url.PathEscape(roomID))
	if err := c.get(topicPath, &topicEvent); err == nil {
		info.Topic = topicEvent.Topic
	}

	return &info, nil
}

// Members returns the joined members of a room with display names.
func (c *Client) Members(roomID string) ([]Member, error) {
	var resp struct {
		Joined map[string]struct {
			DisplayName string `json:"display_name"`
		} `json:"joined"`
	}
	path := fmt.Sprintf("/_matrix/client/v3/rooms/%s/joined_members", url.PathEscape(roomID))
	if err := c.get(path, &resp); err != nil {
		return nil, fmt.Errorf("members: %w", err)
	}

	members := make([]Member, 0, len(resp.Joined))
	for userID, info := range resp.Joined {
		members = append(members, Member{
			UserID:      userID,
			DisplayName: info.DisplayName,
		})
	}
	return members, nil
}

// --- HTTP helpers ---

func (c *Client) post(path string, body interface{}, result interface{}) error {
	return c.do(http.MethodPost, path, body, result)
}

func (c *Client) put(path string, body interface{}, result interface{}) error {
	return c.do(http.MethodPut, path, body, result)
}

func (c *Client) get(path string, result interface{}) error {
	return c.do(http.MethodGet, path, nil, result)
}

type matrixError struct {
	ErrCode string `json:"errcode"`
	Error   string `json:"error"`
}

func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshalling request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(context.Background(), method, c.Homeserver+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var mErr matrixError
		if json.Unmarshal(respBody, &mErr) == nil && mErr.Error != "" {
			return fmt.Errorf("%s (%s)", mErr.Error, mErr.ErrCode)
		}
		return fmt.Errorf("%s %s: status %d", method, path, resp.StatusCode)
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// TokenFromEnv reads the session-scoped Matrix access token from the environment.
func TokenFromEnv() (string, error) {
	token := os.Getenv("JACK_MSG_TOKEN")
	if token == "" {
		return "", fmt.Errorf("JACK_MSG_TOKEN not set (session not configured for messaging)")
	}
	return token, nil
}
