package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	roomsCmd.Flags().StringP("user", "u", "", "username for token lookup (required)")
	_ = roomsCmd.MarkFlagRequired("user")
	Cmd.AddCommand(roomsCmd)
}

var roomsCmd = &cobra.Command{
	Use:   "rooms",
	Short: "List joined rooms",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		username, _ := cmd.Flags().GetString("user")
		token, err := LoadToken(username)
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runRooms(client.JoinedRooms)
	},
}

func runRooms(list RoomLister) error {
	rooms, err := list()
	if err != nil {
		return err
	}
	for _, room := range rooms.Rooms {
		fmt.Println(room)
	}
	return nil
}
