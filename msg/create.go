package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	createCmd.Flags().StringP("user", "u", "", "username for token lookup (required)")
	_ = createCmd.MarkFlagRequired("user")
	Cmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		username, _ := cmd.Flags().GetString("user")
		token, err := LoadToken(username)
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runCreate(args[0], client.CreateRoom)
	},
}

func runCreate(name string, create RoomCreator) error {
	room, err := create(name)
	if err != nil {
		return err
	}
	fmt.Printf("created room %s\n", room.RoomID)
	return nil
}
