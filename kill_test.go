//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunKillNotFound(t *testing.T) {
	err := runKill("blue-vicky", noopChecker, func(_ string) error { return nil })
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "not found"), true)
}

func TestRunKillSuccess(t *testing.T) {
	var killed string
	err := runKill("blue-vicky", existsChecker, func(name string) error {
		killed = name
		return nil
	})
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, killed, "blue-vicky")
}
