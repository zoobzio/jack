//go:build testing

package msg

import (
	"context"
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunWatchSuccess(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, roomID string) (*SyncResponse, error) {
		jtesting.AssertEqual(t, roomID, "") // no room filter
		callCount++
		if callCount == 1 {
			return &SyncResponse{NextBatch: "batch_1"}, nil
		}
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!board:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hello"}, EventID: "$evt1"},
							},
						},
					},
					"!dm:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@bob:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hey"}, EventID: "$evt2"},
							},
						},
					},
				},
			},
		}, nil
	}
	getInfo := func(roomID string) (*RoomInfo, error) {
		return &RoomInfo{Name: "test-room"}, nil
	}
	err := runWatch(5, false, false, syncer, getInfo)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 2)
}

func TestRunWatchNoMessages(t *testing.T) {
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{NextBatch: "batch_1"}, nil
	}
	err := runWatch(1, false, false, syncer, nil)
	jtesting.AssertNoError(t, err)
}

func TestRunWatchFollow(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			return &SyncResponse{NextBatch: "batch_1"}, nil
		}
		if callCount <= 3 {
			return &SyncResponse{NextBatch: fmt.Sprintf("batch_%d", callCount)}, nil
		}
		return nil, fmt.Errorf("done")
	}
	// follow keeps looping through empty syncs until a real error
	err := runWatch(0, true, false, syncer, nil)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, callCount, 4)
}

func TestRunWatchInvite(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			return &SyncResponse{NextBatch: "batch_1"}, nil
		}
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Invite: map[string]SyncInvitedRoom{
					"!newroom:localhost": {
						InviteState: SyncInviteState{
							Events: []Message{
								{Type: "m.room.name", Content: map[string]interface{}{"name": "planning"}},
								{Type: "m.room.member", Sender: "@alice:localhost", Content: map[string]interface{}{"membership": "invite"}},
							},
						},
					},
				},
			},
		}, nil
	}
	err := runWatch(5, false, false, syncer, nil)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 2)
}

func TestRunWatchInviteJSON(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			return &SyncResponse{NextBatch: "batch_1"}, nil
		}
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Invite: map[string]SyncInvitedRoom{
					"!newroom:localhost": {
						InviteState: SyncInviteState{
							Events: []Message{
								{Type: "m.room.name", Content: map[string]interface{}{"name": "planning"}},
								{Type: "m.room.member", Sender: "@alice:localhost", Content: map[string]interface{}{"membership": "invite"}},
							},
						},
					},
				},
			},
		}, nil
	}
	err := runWatch(5, false, true, syncer, nil)
	jtesting.AssertNoError(t, err)
}

func TestRunWatchJSON(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			return &SyncResponse{NextBatch: "batch_1"}, nil
		}
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!room:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "test"}, EventID: "$evt1"},
							},
						},
					},
				},
			},
		}, nil
	}
	err := runWatch(5, false, true, syncer, nil)
	jtesting.AssertNoError(t, err)
}
