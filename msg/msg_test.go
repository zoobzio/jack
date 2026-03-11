//go:build testing

package msg

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestClientRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jtesting.AssertEqual(t, r.URL.Path, "/_matrix/client/v3/register")
		jtesting.AssertEqual(t, r.Method, http.MethodPost)
		json.NewEncoder(w).Encode(Registration{
			UserID:      "@agent:localhost",
			AccessToken: "tok_123",
			DeviceID:    "DEV1",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	reg, err := client.Register("agent", "pass", "jack")
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, reg.UserID, "@agent:localhost")
	jtesting.AssertEqual(t, reg.AccessToken, "tok_123")
}

func TestClientLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jtesting.AssertEqual(t, r.URL.Path, "/_matrix/client/v3/login")
		json.NewEncoder(w).Encode(Registration{
			UserID:      "@operator:localhost",
			AccessToken: "tok_456",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	reg, err := client.Login("operator", "pass")
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, reg.UserID, "@operator:localhost")
	jtesting.AssertEqual(t, reg.AccessToken, "tok_456")
}

func TestClientCreateRoom(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jtesting.AssertEqual(t, r.URL.Path, "/_matrix/client/v3/createRoom")
		jtesting.AssertEqual(t, r.Header.Get("Authorization"), "Bearer tok_abc")
		json.NewEncoder(w).Encode(Room{RoomID: "!room123:localhost"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok_abc")
	room, err := client.CreateRoom("general", "dev discussion")
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, room.RoomID, "!room123:localhost")
}

func TestClientSend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jtesting.AssertEqual(t, r.Method, http.MethodPut)
		json.NewEncoder(w).Encode(map[string]string{"event_id": "$evt1"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok_abc")
	eventID, err := client.Send("!room:localhost", "hello")
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, eventID, "$evt1")
}

func TestClientMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jtesting.AssertEqual(t, r.Method, http.MethodGet)
		json.NewEncoder(w).Encode(Messages{
			Chunk: []Message{
				{Sender: "@bob:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hi"}},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok_abc")
	msgs, err := client.Messages("!room:localhost", 10)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(msgs.Chunk), 1)
	jtesting.AssertEqual(t, msgs.Chunk[0].Sender, "@bob:localhost")
}

func TestClientJoinedRooms(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(JoinedRooms{Rooms: []string{"!a:localhost", "!b:localhost"}})
	}))
	defer server.Close()

	client := NewClient(server.URL, "tok_abc")
	rooms, err := client.JoinedRooms()
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(rooms.Rooms), 2)
}

func TestClientErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(matrixError{ErrCode: "M_FORBIDDEN", Error: "not allowed"})
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	_, err := client.Login("x", "y")
	jtesting.AssertError(t, err)
}

func TestTokenFromEnv(t *testing.T) {
	t.Setenv("JACK_MSG_TOKEN", "tok_env")
	token, err := TokenFromEnv()
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, token, "tok_env")
}

func TestTokenFromEnvMissing(t *testing.T) {
	t.Setenv("JACK_MSG_TOKEN", "")
	_, err := TokenFromEnv()
	jtesting.AssertError(t, err)
}
