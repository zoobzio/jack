//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunRoomsSuccess(t *testing.T) {
	lister := func() (*JoinedRooms, error) {
		return &JoinedRooms{Rooms: []string{"!a:localhost", "!b:localhost"}}, nil
	}
	err := runRooms(lister)
	jtesting.AssertNoError(t, err)
}

func TestRunRoomsError(t *testing.T) {
	lister := func() (*JoinedRooms, error) {
		return nil, fmt.Errorf("unauthorized")
	}
	err := runRooms(lister)
	jtesting.AssertError(t, err)
}
