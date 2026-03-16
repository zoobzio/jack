package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(leaveCmd)
}

var leaveCmd = &cobra.Command{
	Use:   "leave <room>",
	Short: "Leave a room",
	Long:  "Leave a room. The room argument can be a room ID, alias, or short alias name.",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		roomID, err := ResolveRoomID(args[0], client.ResolveAlias)
		if err != nil {
			return err
		}
		return runLeave(roomID, client.Leave)
	},
}

func runLeave(roomID string, leave RoomLeaver) error {
	if err := leave(roomID); err != nil {
		return err
	}
	fmt.Printf("left %s\n", roomID)
	return nil
}
