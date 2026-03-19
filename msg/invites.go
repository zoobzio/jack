package msg

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	invitesCmd.Flags().Bool("accept", false, "auto-join all pending invites")
	Cmd.AddCommand(invitesCmd)
}

var invitesCmd = &cobra.Command{
	Use:   "invites",
	Short: "List pending room invites",
	Long:  "Show rooms you have been invited to but not yet joined.\nUse --accept to auto-join all pending invites.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		accept, _ := cmd.Flags().GetBool("accept")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runInvites(accept, client.Sync, client.Join, client.GetRoomInfo)
	},
}

// inviteInfo summarises a pending invite for display.
type inviteInfo struct {
	RoomID string
	Name   string
	Sender string
}

func runInvites(accept bool, sync syncFunc, join RoomJoiner, _ RoomInfoGetter) error {
	// A single sync with timeout=0 returns current state including invites.
	resp, err := sync(context.Background(), "", 0, "")
	if err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	if len(resp.Rooms.Invite) == 0 {
		fmt.Println("no pending invites")
		return nil
	}

	invites := parseInvites(resp.Rooms.Invite)

	for _, inv := range invites {
		name := inv.Name
		if name == "" {
			name = inv.RoomID
		}
		if accept {
			if _, joinErr := join(inv.RoomID); joinErr != nil {
				fmt.Printf("%-50s  FAILED  %v\n", name, joinErr)
				continue
			}
			fmt.Printf("%-50s  joined\n", name)
		} else {
			sender := inv.Sender
			if sender == "" {
				sender = unknownPlaceholder
			}
			fmt.Printf("%-50s  from %s\n", name, sender)
		}
	}

	return nil
}
