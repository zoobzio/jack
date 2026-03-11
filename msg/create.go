package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	createCmd.Flags().String("topic", "", "room topic describing its purpose")
	Cmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a room",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		topic, _ := cmd.Flags().GetString("topic")
		client := NewClient(Homeserver, token)
		return runCreate(args[0], topic, client.CreateRoom)
	},
}

func runCreate(name, topic string, create RoomCreator) error {
	room, err := create(name, topic)
	if err != nil {
		return err
	}
	fmt.Printf("created room %s\n", room.RoomID)
	return nil
}
