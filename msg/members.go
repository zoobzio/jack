package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(membersCmd)
}

var membersCmd = &cobra.Command{
	Use:   "members <room-id>",
	Short: "List members of a room",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runMembers(args[0], client.Members)
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
