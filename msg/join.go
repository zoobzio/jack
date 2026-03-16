package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(joinCmd)
}

var joinCmd = &cobra.Command{
	Use:   "join <room-id-or-alias>",
	Short: "Join a room",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runJoin(args[0], client.ResolveAlias, client.Join)
	},
}

func runJoin(roomIDOrAlias string, resolve AliasResolver, join RoomJoiner) error {
	resolved, err := ResolveRoomID(roomIDOrAlias, resolve)
	if err != nil {
		return err
	}
	roomID, err := join(resolved)
	if err != nil {
		return err
	}
	fmt.Printf("joined %s\n", roomID)
	return nil
}
