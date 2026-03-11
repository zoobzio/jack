package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(inviteCmd)
}

var inviteCmd = &cobra.Command{
	Use:   "invite <room-id> <user-id>",
	Short: "Invite a user to a room",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runInvite(args[0], args[1], client.Invite)
	},
}

func runInvite(roomID, userID string, invite Inviter) error {
	if err := invite(roomID, userID); err != nil {
		return err
	}
	fmt.Printf("invited %s to %s\n", userID, roomID)
	return nil
}
