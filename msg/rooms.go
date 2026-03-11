package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(roomsCmd)
}

var roomsCmd = &cobra.Command{
	Use:   "rooms",
	Short: "List joined rooms",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runRooms(client.JoinedRooms, client.GetRoomInfo)
	},
}

func runRooms(list RoomLister, getInfo RoomInfoGetter) error {
	rooms, err := list()
	if err != nil {
		return err
	}
	for _, roomID := range rooms.Rooms {
		info, err := getInfo(roomID)
		if err != nil {
			fmt.Println(roomID)
			continue
		}
		name := info.Name
		if name == "" {
			name = roomID
		}
		if info.Topic != "" {
			fmt.Printf("%s  %s  %s\n", roomID, name, info.Topic)
		} else {
			fmt.Printf("%s  %s\n", roomID, name)
		}
	}
	return nil
}
