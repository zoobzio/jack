//go:build testing

package msg

import (
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunNoteAddSuccess(t *testing.T) {
	Homeserver = "http://localhost:8008"
	var sentRoom, sentMsg string
	sender := func(roomID, message string) (string, error) {
		sentRoom = roomID
		sentMsg = message
		return "$note1", nil
	}
	err := runNoteAdd("note-blue", "Notes for agent blue", "note-blue", "remember this", stubResolver("!note:localhost"), sender, stubCreator("!note:localhost"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, sentRoom, "!note:localhost")
	jtesting.AssertEqual(t, sentMsg, "remember this")
}

func TestRunNoteAddCreatesRoom(t *testing.T) {
	Homeserver = "http://localhost:8008"
	var created bool
	creator := func(_, _, _ string) (*Room, error) {
		created = true
		return &Room{RoomID: "!new:localhost"}, nil
	}
	sender := func(_, _ string) (string, error) { return "$note1", nil }
	err := runNoteAdd("note-blue", "Notes for agent blue", "note-blue", "first note", failResolver(), sender, creator)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, created, true)
}

func TestRunNoteListPendingOnly(t *testing.T) {
	Homeserver = "http://localhost:8008"
	reader := func(_ string, _ int) (*Messages, error) {
		return &Messages{
			Chunk: []Message{
				// Reverse chronological: newest first.
				{EventID: "$n2", Sender: "@a:localhost", Type: msgTypeRoomMessage, Content: map[string]interface{}{"body": "second"}},
				{EventID: "$r1", Sender: "@a:localhost", Type: msgTypeReaction, Content: map[string]interface{}{
					"m.relates_to": map[string]interface{}{
						"rel_type": "m.annotation",
						"event_id": "$n1",
						"key":      doneReactionKey,
					},
				}},
				{EventID: "$n1", Sender: "@a:localhost", Type: msgTypeRoomMessage, Content: map[string]interface{}{"body": "first"}},
			},
		}, nil
	}
	// Default (pending only) — $n1 is done, so only $n2 should appear.
	err := runNoteList("note-blue", "Notes for agent blue", "note-blue", 50, false, false, stubResolver("!note:localhost"), reader, stubCreator("!note:localhost"))
	jtesting.AssertNoError(t, err)
}

func TestRunNoteListAll(t *testing.T) {
	Homeserver = "http://localhost:8008"
	reader := func(_ string, _ int) (*Messages, error) {
		return &Messages{
			Chunk: []Message{
				{EventID: "$n2", Sender: "@a:localhost", Type: msgTypeRoomMessage, Content: map[string]interface{}{"body": "second"}},
				{EventID: "$r1", Sender: "@a:localhost", Type: msgTypeReaction, Content: map[string]interface{}{
					"m.relates_to": map[string]interface{}{
						"rel_type": "m.annotation",
						"event_id": "$n1",
						"key":      doneReactionKey,
					},
				}},
				{EventID: "$n1", Sender: "@a:localhost", Type: msgTypeRoomMessage, Content: map[string]interface{}{"body": "first"}},
			},
		}, nil
	}
	// --all shows both notes.
	err := runNoteList("note-blue", "Notes for agent blue", "note-blue", 50, true, false, stubResolver("!note:localhost"), reader, stubCreator("!note:localhost"))
	jtesting.AssertNoError(t, err)
}

func TestRunNoteListJSON(t *testing.T) {
	Homeserver = "http://localhost:8008"
	reader := func(_ string, _ int) (*Messages, error) {
		return &Messages{
			Chunk: []Message{
				{EventID: "$n1", Sender: "@a:localhost", Type: msgTypeRoomMessage, Content: map[string]interface{}{"body": "todo"}},
			},
		}, nil
	}
	err := runNoteList("note-blue", "Notes for agent blue", "note-blue", 50, false, true, stubResolver("!note:localhost"), reader, stubCreator("!note:localhost"))
	jtesting.AssertNoError(t, err)
}

func TestRunNoteDoneSuccess(t *testing.T) {
	Homeserver = "http://localhost:8008"
	var reactedRoom, reactedEvent, reactedKey string
	reactor := func(roomID, eventID, key string) (string, error) {
		reactedRoom = roomID
		reactedEvent = eventID
		reactedKey = key
		return "$react1", nil
	}
	err := runNoteDone("note-blue", "Notes for agent blue", "note-blue", "$n1", stubResolver("!note:localhost"), reactor, stubCreator("!note:localhost"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, reactedRoom, "!note:localhost")
	jtesting.AssertEqual(t, reactedEvent, "$n1")
	jtesting.AssertEqual(t, reactedKey, doneReactionKey)
}
