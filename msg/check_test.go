//go:build testing

package msg

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunCheckPendingMessages(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, roomID string) (*SyncResponse, error) {
		callCount++
		jtesting.AssertEqual(t, since, "saved_batch")
		jtesting.AssertEqual(t, timeout, 0)
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!room:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "pending msg"}, EventID: "$evt1"},
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
	var savedToken string
	load := func() string { return "saved_batch" }
	save := func(token string) error { savedToken = token; return nil }

	err := runCheck(0, false, syncer, getInfo, load, save)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 1)
	jtesting.AssertEqual(t, savedToken, "batch_2")
}

func TestRunCheckPendingMessagesJSON(t *testing.T) {
	syncer := func(_ context.Context, since string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!room:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hello"}, EventID: "$evt1"},
							},
						},
					},
				},
			},
		}, nil
	}
	load := func() string { return "saved_batch" }
	save := func(_ string) error { return nil }

	err := runCheck(0, true, syncer, nil, load, save)
	jtesting.AssertNoError(t, err)
}

func TestRunCheckNoTokenFreshSync(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			// Initial sync — no saved token.
			jtesting.AssertEqual(t, since, "")
			jtesting.AssertEqual(t, timeout, 0)
			return &SyncResponse{NextBatch: "initial_batch"}, nil
		}
		// Watch mode — returns a message to break the loop.
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!room:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@bob:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "new msg"}},
							},
						},
					},
				},
			},
		}, nil
	}
	var tokens []string
	load := func() string { return "" }
	save := func(token string) error { tokens = append(tokens, token); return nil }

	err := runCheck(0, false, syncer, nil, load, save)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 2)
	// Should have saved twice: initial sync + watch iteration.
	jtesting.AssertEqual(t, len(tokens), 2)
	jtesting.AssertEqual(t, tokens[0], "initial_batch")
	jtesting.AssertEqual(t, tokens[1], "batch_2")
}

func TestRunCheckNoPendingEntersWatch(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			// Immediate sync with saved token — no messages.
			jtesting.AssertEqual(t, since, "saved_batch")
			jtesting.AssertEqual(t, timeout, 0)
			return &SyncResponse{NextBatch: "batch_2"}, nil
		}
		// Watch mode poll — returns a message to exit.
		jtesting.AssertEqual(t, since, "batch_2")
		jtesting.AssertEqual(t, timeout, pollInterval)
		return &SyncResponse{
			NextBatch: "batch_3",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!room:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "arrived"}},
							},
						},
					},
				},
			},
		}, nil
	}
	load := func() string { return "saved_batch" }
	save := func(_ string) error { return nil }

	err := runCheck(0, false, syncer, nil, load, save)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 2)
}

func TestRunCheckStaleTokenFallback(t *testing.T) {
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, _ string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			// Stale token — server rejects it.
			jtesting.AssertEqual(t, since, "stale_token")
			return nil, fmt.Errorf("M_UNKNOWN_TOKEN")
		}
		if callCount == 2 {
			// Fallback fresh sync.
			jtesting.AssertEqual(t, since, "")
			return &SyncResponse{NextBatch: "fresh_batch"}, nil
		}
		// Watch mode — deliver a message.
		return &SyncResponse{
			NextBatch: "batch_2",
			Rooms: SyncRooms{
				Join: map[string]SyncJoinedRoom{
					"!room:localhost": {
						Timeline: SyncTimeline{
							Events: []Message{
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hi"}},
							},
						},
					},
				},
			},
		}, nil
	}
	load := func() string { return "stale_token" }
	save := func(_ string) error { return nil }

	err := runCheck(0, false, syncer, nil, load, save)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 3)
}

func TestRunCheckTimeout(t *testing.T) {
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{NextBatch: "batch_1"}, nil
	}
	load := func() string { return "" }
	save := func(_ string) error { return nil }

	// With a 1-second timeout, should exit without error when no messages arrive.
	err := runCheck(1, false, syncer, nil, load, save)
	jtesting.AssertNoError(t, err)
}

func TestRunCheckInvite(t *testing.T) {
	syncer := func(_ context.Context, since string, _ int, _ string) (*SyncResponse, error) {
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
	load := func() string { return "saved_batch" }
	save := func(_ string) error { return nil }

	err := runCheck(0, false, syncer, nil, load, save)
	jtesting.AssertNoError(t, err)
}

func TestLoadSaveSyncToken(t *testing.T) {
	dir := t.TempDir()
	jackDir := filepath.Join(dir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	t.Chdir(dir)

	// No token yet.
	token := loadSyncToken()
	jtesting.AssertEqual(t, token, "")

	// Save a token.
	err := saveSyncToken("batch_abc")
	jtesting.AssertNoError(t, err)

	// Load it back.
	token = loadSyncToken()
	jtesting.AssertEqual(t, token, "batch_abc")
}

func TestSaveSyncTokenNoJackDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	err := saveSyncToken("batch_abc")
	jtesting.AssertError(t, err)
}
