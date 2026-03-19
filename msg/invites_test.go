//go:build testing

package msg

import (
	"context"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunInvitesNone(t *testing.T) {
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{NextBatch: "batch_1"}, nil
	}
	err := runInvites(false, syncer, nil, nil)
	jtesting.AssertNoError(t, err)
}

func TestRunInvitesList(t *testing.T) {
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{
			NextBatch: "batch_1",
			Rooms: SyncRooms{
				Invite: map[string]SyncInvitedRoom{
					"!room:localhost": {
						InviteState: SyncInviteState{
							Events: []Message{
								{Type: "m.room.name", Content: map[string]interface{}{"name": "test-room"}},
								{Type: "m.room.member", Sender: "@alice:localhost", Content: map[string]interface{}{"membership": "invite"}},
							},
						},
					},
				},
			},
		}, nil
	}
	err := runInvites(false, syncer, nil, nil)
	jtesting.AssertNoError(t, err)
}

func TestRunInvitesAccept(t *testing.T) {
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{
			NextBatch: "batch_1",
			Rooms: SyncRooms{
				Invite: map[string]SyncInvitedRoom{
					"!room:localhost": {
						InviteState: SyncInviteState{
							Events: []Message{
								{Type: "m.room.name", Content: map[string]interface{}{"name": "test-room"}},
								{Type: "m.room.member", Sender: "@alice:localhost", Content: map[string]interface{}{"membership": "invite"}},
							},
						},
					},
				},
			},
		}, nil
	}
	var joinedRoom string
	joiner := func(roomID string) (string, error) {
		joinedRoom = roomID
		return roomID, nil
	}
	err := runInvites(true, syncer, joiner, nil)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, joinedRoom, "!room:localhost")
}
