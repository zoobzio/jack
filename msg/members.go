package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(membersCmd)
}

var membersCmd = &cobra.Command{
	Use:   "members <room>",
	Short: "List members of a room",
	Long:  "List members of a room. The room argument can be a room ID, alias, or short alias name.",
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
		return runMembers(roomID, client.Members)
	},
}

func runMembers(roomID string, list MemberLister) error {
	members, err := list(roomID)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.DisplayName != "" {
			fmt.Printf("%s  %s\n", m.UserID, m.DisplayName)
		} else {
			fmt.Println(m.UserID)
		}
	}
	return nil
}
