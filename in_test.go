//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunInNotFound(t *testing.T) {
	err := runIn("blue-vicky", noopChecker, func(_ string) error { return nil })
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "not found"), true)
}

func TestRunInSuccess(t *testing.T) {
	var attached string
	err := runIn("blue-vicky", existsChecker, func(name string) error {
		attached = name
		return nil
	})
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attached, "blue-vicky")
}
