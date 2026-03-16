package msg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	watchCmd.Flags().Int("timeout", 30, "seconds to wait before giving up")
	watchCmd.Flags().BoolP("follow", "f", false, "stream messages continuously")
	watchCmd.Flags().Bool("json", false, "output messages as JSON")
	Cmd.AddCommand(watchCmd)
}

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch all joined rooms for new messages",
	Long:  "Block until a new message arrives in any joined room, print it, and exit.\nUse --follow to stream continuously across all rooms.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		timeout, _ := cmd.Flags().GetInt("timeout")
		follow, _ := cmd.Flags().GetBool("follow")
		jsonFlag, _ := cmd.Flags().GetBool("json")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runWatch(timeout, follow, jsonFlag, client.Sync, client.GetRoomInfo)
	},
}

type watchMessage struct {
	RoomID   string `json:"room_id"`
	RoomName string `json:"room_name,omitempty"`
	Sender   string `json:"sender"`
	Body     string `json:"body"`
	EventID  string `json:"event_id"`
}

func runWatch(timeout int, follow, jsonOut bool, sync syncFunc, getInfo RoomInfoGetter) error {
	ctx := context.Background()
	if !follow && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second+5*time.Second)
		defer cancel()
	}

	// Initial sync (no room filter = all rooms).
	resp, err := sync(ctx, "", 0, "")
	if err != nil {
		return fmt.Errorf("initial sync: %w", err)
	}

	// Cache room names.
	roomNames := map[string]string{}
	lookupName := func(roomID string) string {
		if name, ok := roomNames[roomID]; ok {
			return name
		}
		if getInfo != nil {
			if info, getErr := getInfo(roomID); getErr == nil && info.Name != "" {
				roomNames[roomID] = info.Name
				return info.Name
			}
		}
		roomNames[roomID] = roomID
		return roomID
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	for {
		resp, err = sync(ctx, resp.NextBatch, timeout, "")
		if err != nil {
			return fmt.Errorf("sync: %w", err)
		}

		found := false
		for roomID, room := range resp.Rooms.Join {
			for _, m := range room.Timeline.Events {
				if m.Type != msgTypeRoomMessage {
					continue
				}
				found = true
				body, _ := m.Content["body"].(string)
				if jsonOut {
					if err := enc.Encode(watchMessage{
						RoomID:   roomID,
						RoomName: lookupName(roomID),
						Sender:   m.Sender,
						Body:     body,
						EventID:  m.EventID,
					}); err != nil {
						return fmt.Errorf("encoding message: %w", err)
					}
				} else {
					fmt.Printf("[%s] %s: %s\n", lookupName(roomID), m.Sender, body)
				}
			}
		}

		if !follow {
			if !found {
				return fmt.Errorf("no new messages within timeout")
			}
			return nil
		}
	}
}
