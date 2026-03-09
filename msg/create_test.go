//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunCreateSuccess(t *testing.T) {
	var createdName string
	creator := func(name string) (*Room, error) {
		createdName = name
		return &Room{RoomID: "!new:localhost"}, nil
	}
	err := runCreate("general", creator)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, createdName, "general")
}

func TestRunCreateError(t *testing.T) {
	creator := func(_ string) (*Room, error) {
		return nil, fmt.Errorf("already exists")
	}
	err := runCreate("general", creator)
	jtesting.AssertError(t, err)
}
