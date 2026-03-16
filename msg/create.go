package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	createCmd.Flags().String("topic", "", "room topic describing its purpose")
	createCmd.Flags().String("alias", "", "canonical room alias (e.g. general creates #general:server)")
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
		alias, _ := cmd.Flags().GetString("alias")
		client := NewClient(Homeserver, token)
		if alias != "" {
			return runCreateWithAlias(args[0], topic, alias, client.CreateRoomWithAlias)
		}
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

type aliasRoomCreator func(name, topic, alias string) (*Room, error)

func runCreateWithAlias(name, topic, alias string, create aliasRoomCreator) error {
	room, err := create(name, topic, alias)
	if err != nil {
		return err
	}
	fmt.Printf("created room %s (#%s:%s)\n", room.RoomID, alias, ServerName(Homeserver))
	return nil
}
