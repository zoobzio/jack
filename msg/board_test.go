//go:build testing

package msg

import (
	"context"
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func stubResolver(roomID string) AliasResolver {
	return func(_ string) (*AliasResponse, error) {
		return &AliasResponse{RoomID: roomID}, nil
	}
}

func failResolver() AliasResolver {
	return func(_ string) (*AliasResponse, error) {
		return nil, fmt.Errorf("not found")
	}
}

func stubCreator(roomID string) func(string, string, string) (*Room, error) {
	return func(_, _, _ string) (*Room, error) {
		return &Room{RoomID: roomID}, nil
	}
}

func TestRunBoardPostSuccess(t *testing.T) {
	Homeserver = "http://localhost:8008"
	var sentRoom, sentMsg string
	sender := func(roomID, message string) (string, error) {
		sentRoom = roomID
		sentMsg = message
		return "$evt1", nil
	}
	err := runBoardPost("board-blue", "Construct board for team blue", "board-blue", "hello board", stubResolver("!board:localhost"), sender, stubCreator("!board:localhost"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, sentRoom, "!board:localhost")
	jtesting.AssertEqual(t, sentMsg, "hello board")
}

func TestRunBoardPostCreatesRoom(t *testing.T) {
	Homeserver = "http://localhost:8008"
	var created bool
	creator := func(name, topic, alias string) (*Room, error) {
		created = true
		return &Room{RoomID: "!new:localhost"}, nil
	}
	sender := func(_, _ string) (string, error) { return "$evt1", nil }
	err := runBoardPost("board-blue", "Construct board for team blue", "board-blue", "first post", failResolver(), sender, creator)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, created, true)
}

func TestRunBoardReadSuccess(t *testing.T) {
	Homeserver = "http://localhost:8008"
	reader := func(roomID string, limit int) (*Messages, error) {
		return &Messages{
			Chunk: []Message{
				{Sender: "@bob:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hey"}},
			},
		}, nil
	}
	err := runBoardRead("board-blue", "Construct board for team blue", "board-blue", 10, false, "", stubResolver("!board:localhost"), reader, stubCreator("!board:localhost"))
	jtesting.AssertNoError(t, err)
}

func TestRunBoardReadJSON(t *testing.T) {
	Homeserver = "http://localhost:8008"
	reader := func(roomID string, limit int) (*Messages, error) {
		return &Messages{
			Chunk: []Message{
				{Sender: "@bob:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hey"}, EventID: "$evt1"},
			},
		}, nil
	}
	err := runBoardRead("board-blue", "Construct board for team blue", "board-blue", 10, true, "", stubResolver("!board:localhost"), reader, stubCreator("!board:localhost"))
	jtesting.AssertNoError(t, err)
}

func TestRunBoardWatchSuccess(t *testing.T) {
	Homeserver = "http://localhost:8008"
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, roomID string) (*SyncResponse, error) {
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
								{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "new message"}},
							},
						},
					},
				},
			},
		}, nil
	}
	err := runBoardWatch("board-blue", "Construct board for team blue", "board-blue", 5, false, stubResolver("!board:localhost"), syncer, stubCreator("!board:localhost"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, callCount, 2)
}

func TestRunBoardWatchNoMessages(t *testing.T) {
	Homeserver = "http://localhost:8008"
	syncer := func(_ context.Context, _ string, _ int, _ string) (*SyncResponse, error) {
		return &SyncResponse{NextBatch: "batch_1"}, nil
	}
	err := runBoardWatch("board-blue", "Construct board for team blue", "board-blue", 1, false, stubResolver("!board:localhost"), syncer, stubCreator("!board:localhost"))
	jtesting.AssertError(t, err)
}

func TestRunBoardWatchFollow(t *testing.T) {
	Homeserver = "http://localhost:8008"
	callCount := 0
	syncer := func(_ context.Context, since string, timeout int, roomID string) (*SyncResponse, error) {
		callCount++
		if callCount == 1 {
			// Initial sync.
			return &SyncResponse{NextBatch: "batch_1"}, nil
		}
		if callCount <= 3 {
			// Empty syncs (follow should continue).
			return &SyncResponse{NextBatch: fmt.Sprintf("batch_%d", callCount)}, nil
		}
		// Return an error to break the loop for testing.
		return nil, fmt.Errorf("done")
	}
	err := runBoardWatch("board-blue", "Construct board for team blue", "board-blue", 1, true, stubResolver("!board:localhost"), syncer, stubCreator("!board:localhost"))
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, callCount, 4)
}
