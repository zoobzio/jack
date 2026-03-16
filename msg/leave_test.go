//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunLeaveSuccess(t *testing.T) {
	var leftRoom string
	leaver := func(roomID string) error {
		leftRoom = roomID
		return nil
	}
	err := runLeave("!room:localhost", leaver)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, leftRoom, "!room:localhost")
}

func TestRunLeaveError(t *testing.T) {
	leaver := func(_ string) error {
		return fmt.Errorf("forbidden")
	}
	err := runLeave("!room:localhost", leaver)
	jtesting.AssertError(t, err)
}
