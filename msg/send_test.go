//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunSendSuccess(t *testing.T) {
	var sentRoom, sentMsg string
	sender := func(roomID, message string) (string, error) {
		sentRoom = roomID
		sentMsg = message
		return "$evt1", nil
	}
	err := runSend("!room:localhost", "hello world", sender)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, sentRoom, "!room:localhost")
	jtesting.AssertEqual(t, sentMsg, "hello world")
}

func TestRunSendError(t *testing.T) {
	sender := func(_, _ string) (string, error) {
		return "", fmt.Errorf("send failed")
	}
	err := runSend("!room:localhost", "hello", sender)
	jtesting.AssertError(t, err)
}
