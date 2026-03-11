//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func noopInfoGetter(_ string) (*RoomInfo, error) {
	return &RoomInfo{}, nil
}

func TestRunRoomsSuccess(t *testing.T) {
	lister := func() (*JoinedRooms, error) {
		return &JoinedRooms{Rooms: []string{"!a:localhost", "!b:localhost"}}, nil
	}
	infoGetter := func(roomID string) (*RoomInfo, error) {
		return &RoomInfo{Name: "test-room", Topic: "a topic"}, nil
	}
	err := runRooms(lister, infoGetter)
	jtesting.AssertNoError(t, err)
}

func TestRunRoomsError(t *testing.T) {
	lister := func() (*JoinedRooms, error) {
		return nil, fmt.Errorf("unauthorized")
	}
	err := runRooms(lister, noopInfoGetter)
	jtesting.AssertError(t, err)
}
