package msg

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	sendCmd.Flags().Bool("json", false, "validate message is valid JSON before sending")
	sendCmd.Flags().Bool("stdin", false, "read message from stdin")
	Cmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:   "send <room> <message...>",
	Short: "Send a message to a room",
	Long:  "Send a message to a room. The room argument can be a room ID, alias, or short alias name.\nUse '-' as the message or --stdin to read from stdin.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		roomID, err := ResolveRoomID(args[0], client.ResolveAlias)
		if err != nil {
			return err
		}

		stdinFlag, _ := cmd.Flags().GetBool("stdin")
		var message string
		if stdinFlag || (len(args) == 2 && args[1] == "-") {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			message = strings.TrimRight(string(data), "\n")
		} else {
			if len(args) < 2 {
				return fmt.Errorf("message required (use --stdin to read from stdin)")
			}
			message = strings.Join(args[1:], " ")
		}

		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			if !json.Valid([]byte(message)) {
				return fmt.Errorf("message is not valid JSON")
			}
		}
		return runSend(roomID, message, client.Send)
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
