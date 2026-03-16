package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(inviteCmd)
}

var inviteCmd = &cobra.Command{
	Use:   "invite <room> <user-id>",
	Short: "Invite a user to a room",
	Long:  "Invite a user to a room. The room argument can be a room ID, alias, or short alias name.",
	Args:  cobra.ExactArgs(2),
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
		return runInvite(roomID, args[1], client.Invite)
	},
}

func runInvite(roomID, userID string, invite Inviter) error {
	if err := invite(roomID, userID); err != nil {
		return err
	}
	fmt.Printf("invited %s to %s\n", userID, roomID)
	return nil
}
