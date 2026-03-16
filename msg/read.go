package msg

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	readCmd.Flags().IntP("limit", "n", 20, "number of messages to retrieve")
	readCmd.Flags().Bool("json", false, "output messages as JSON")
	readCmd.Flags().String("since", "", "show messages after this event ID")
	readCmd.Flags().String("from", "", "filter messages by sender username")
	Cmd.AddCommand(readCmd)
}

var readCmd = &cobra.Command{
	Use:   "read <room>",
	Short: "Read messages from a room",
	Long:  "Read messages from a room. The room argument can be a room ID, alias, or short alias name.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		jsonFlag, _ := cmd.Flags().GetBool("json")
		since, _ := cmd.Flags().GetString("since")
		from, _ := cmd.Flags().GetString("from")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		roomID, err := ResolveRoomID(args[0], client.ResolveAlias)
		if err != nil {
			return err
		}
		if since != "" {
			return runReadSince(roomID, since, limit, jsonFlag, client.EventContext, client.MessagesFrom)
		}
		if from != "" {
			return runReadFiltered(roomID, limit, jsonFlag, from, client.Messages)
		}
		if jsonFlag {
			return runReadJSON(roomID, limit, client.Messages)
		}
		return runRead(roomID, limit, client.Messages)
	},
}

func runRead(roomID string, limit int, read MessageReader) error {
	msgs, err := read(roomID, limit)
	if err != nil {
		return err
	}
	for i := len(msgs.Chunk) - 1; i >= 0; i-- {
		m := msgs.Chunk[i]
		if m.Type != msgTypeRoomMessage {
			continue
		}
		body, _ := m.Content["body"].(string)
		fmt.Printf("%s: %s\n", m.Sender, body)
	}
	return nil
}

type jsonMessage struct {
	Sender  string `json:"sender"`
	Body    string `json:"body"`
	EventID string `json:"event_id"`
}

func runReadJSON(roomID string, limit int, read MessageReader) error {
	msgs, err := read(roomID, limit)
	if err != nil {
		return err
	}
	var out []jsonMessage
	for i := len(msgs.Chunk) - 1; i >= 0; i-- {
		m := msgs.Chunk[i]
		if m.Type != msgTypeRoomMessage {
			continue
		}
		body, _ := m.Content["body"].(string)
		out = append(out, jsonMessage{
			Sender:  m.Sender,
			Body:    body,
			EventID: m.EventID,
		})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// senderMatches checks if a message sender matches the filter.
// Accepts full user IDs (@user:server), bare usernames, or partial matches.
func senderMatches(sender, filter string) bool {
	return strings.Contains(sender, filter)
}

func runReadFiltered(roomID string, limit int, jsonOut bool, from string, read MessageReader) error {
	msgs, err := read(roomID, limit)
	if err != nil {
		return err
	}
	if jsonOut {
		var out []jsonMessage
		for i := len(msgs.Chunk) - 1; i >= 0; i-- {
			m := msgs.Chunk[i]
			if m.Type != msgTypeRoomMessage || !senderMatches(m.Sender, from) {
				continue
			}
			body, _ := m.Content["body"].(string)
			out = append(out, jsonMessage{
				Sender:  m.Sender,
				Body:    body,
				EventID: m.EventID,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	for i := len(msgs.Chunk) - 1; i >= 0; i-- {
		m := msgs.Chunk[i]
		if m.Type != msgTypeRoomMessage || !senderMatches(m.Sender, from) {
			continue
		}
		body, _ := m.Content["body"].(string)
		fmt.Printf("%s: %s\n", m.Sender, body)
	}
	return nil
}

func runReadSince(roomID, eventID string, limit int, jsonOut bool, getContext EventContextGetter, readFrom MessageFromReader) error {
	token, err := getContext(roomID, eventID)
	if err != nil {
		return fmt.Errorf("resolving event %s: %w", eventID, err)
	}
	msgs, err := readFrom(roomID, token, limit, "f")
	if err != nil {
		return err
	}
	// Forward direction returns chronological order — no reversal needed.
	if jsonOut {
		var out []jsonMessage
		for _, m := range msgs.Chunk {
			if m.Type != msgTypeRoomMessage {
				continue
			}
			body, _ := m.Content["body"].(string)
			out = append(out, jsonMessage{
				Sender:  m.Sender,
				Body:    body,
				EventID: m.EventID,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	for _, m := range msgs.Chunk {
		if m.Type != msgTypeRoomMessage {
			continue
		}
		body, _ := m.Content["body"].(string)
		fmt.Printf("%s: %s\n", m.Sender, body)
	}
	return nil
}
