package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	roomsCmd.Flags().Bool("all", false, "show all public rooms on the server")
	Cmd.AddCommand(roomsCmd)
}

var roomsCmd = &cobra.Command{
	Use:   "rooms",
	Short: "List joined rooms",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		all, _ := cmd.Flags().GetBool("all")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		if all {
			return runPublicRooms(client.PublicRooms)
		}
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

func runPublicRooms(list PublicRoomLister) error {
	resp, err := list()
	if err != nil {
		return err
	}
	for _, room := range resp.Chunk {
		name := room.Name
		if name == "" {
			name = room.RoomID
		}
		if room.Topic != "" {
			fmt.Printf("%s  %s  %s  (%d members)\n", room.RoomID, name, room.Topic, room.NumJoined)
		} else {
			fmt.Printf("%s  %s  (%d members)\n", room.RoomID, name, room.NumJoined)
		}
	}
	return nil
}
