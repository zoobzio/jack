//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunInviteSuccess(t *testing.T) {
	var invitedRoom, invitedUser string
	inviter := func(roomID, userID string) error {
		invitedRoom = roomID
		invitedUser = userID
		return nil
	}
	err := runInvite("!room:localhost", "@agent:localhost", inviter)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, invitedRoom, "!room:localhost")
	jtesting.AssertEqual(t, invitedUser, "@agent:localhost")
}

func TestRunInviteError(t *testing.T) {
	inviter := func(_, _ string) error {
		return fmt.Errorf("forbidden")
	}
	err := runInvite("!room:localhost", "@agent:localhost", inviter)
	jtesting.AssertError(t, err)
}
