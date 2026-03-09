package msg

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	sendCmd.Flags().StringP("user", "u", "", "username for token lookup (required)")
	_ = sendCmd.MarkFlagRequired("user")
	Cmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:   "send <room-id> <message...>",
	Short: "Send a message to a room",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("user")
		token, err := LoadToken(username)
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		message := strings.Join(args[1:], " ")
		return runSend(args[0], message, client.Send)
	},
}

func runSend(roomID, message string, send MessageSender) error {
	eventID, err := send(roomID, message)
	if err != nil {
		return err
	}
	fmt.Println(eventID)
	return nil
}
