//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunCreateSuccess(t *testing.T) {
	var createdName, createdTopic string
	creator := func(name, topic string) (*Room, error) {
		createdName = name
		createdTopic = topic
		return &Room{RoomID: "!new:localhost"}, nil
	}
	err := runCreate("general", "dev discussion", creator)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, createdName, "general")
	jtesting.AssertEqual(t, createdTopic, "dev discussion")
}

func TestRunCreateError(t *testing.T) {
	creator := func(_, _ string) (*Room, error) {
		return nil, fmt.Errorf("already exists")
	}
	err := runCreate("general", "", creator)
	jtesting.AssertError(t, err)
}
