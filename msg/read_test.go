//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunReadSuccess(t *testing.T) {
	reader := func(roomID string, limit int) (*Messages, error) {
		jtesting.AssertEqual(t, roomID, "!room:localhost")
		jtesting.AssertEqual(t, limit, 10)
		return &Messages{
			Chunk: []Message{
				{Sender: "@bob:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "second"}},
				{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "first"}},
			},
		}, nil
	}
	err := runRead("!room:localhost", 10, reader)
	jtesting.AssertNoError(t, err)
}

func TestRunReadError(t *testing.T) {
	reader := func(_ string, _ int) (*Messages, error) {
		return nil, fmt.Errorf("not found")
	}
	err := runRead("!room:localhost", 10, reader)
	jtesting.AssertError(t, err)
}

func TestRunReadFiltersNonMessages(t *testing.T) {
	reader := func(_ string, _ int) (*Messages, error) {
		return &Messages{
			Chunk: []Message{
				{Sender: "@bob:localhost", Type: "m.room.member", Content: map[string]interface{}{}},
				{Sender: "@alice:localhost", Type: "m.room.message", Content: map[string]interface{}{"body": "hello"}},
			},
		}, nil
	}
	// Should not error; non-message events are silently skipped.
	err := runRead("!room:localhost", 10, reader)
	jtesting.AssertNoError(t, err)
}
