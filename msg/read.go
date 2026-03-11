package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	readCmd.Flags().IntP("limit", "n", 20, "number of messages to retrieve")
	Cmd.AddCommand(readCmd)
}

var readCmd = &cobra.Command{
	Use:   "read <room-id>",
	Short: "Read messages from a room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runRead(args[0], limit, client.Messages)
	},
}

func runRead(roomID string, limit int, read MessageReader) error {
	msgs, err := read(roomID, limit)
	if err != nil {
		return err
	}
	for i := len(msgs.Chunk) - 1; i >= 0; i-- {
		m := msgs.Chunk[i]
		if m.Type != "m.room.message" {
			continue
		}
		body, _ := m.Content["body"].(string)
		fmt.Printf("%s: %s\n", m.Sender, body)
	}
	return nil
}
