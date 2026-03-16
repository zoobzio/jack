//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunJoinSuccess(t *testing.T) {
	resolver := func(_ string) (*AliasResponse, error) {
		t.Fatal("resolver should not be called for room IDs")
		return nil, nil
	}
	joiner := func(roomIDOrAlias string) (string, error) {
		return "!room:localhost", nil
	}
	err := runJoin("!room:localhost", resolver, joiner)
	jtesting.AssertNoError(t, err)
}

func TestRunJoinError(t *testing.T) {
	resolver := func(_ string) (*AliasResponse, error) {
		t.Fatal("resolver should not be called for room IDs")
		return nil, nil
	}
	joiner := func(_ string) (string, error) {
		return "", fmt.Errorf("forbidden")
	}
	err := runJoin("!room:localhost", resolver, joiner)
	jtesting.AssertError(t, err)
}

func TestRunJoinShortAlias(t *testing.T) {
	Homeserver = "http://localhost:8008"
	var resolvedAlias string
	resolver := func(alias string) (*AliasResponse, error) {
		resolvedAlias = alias
		return &AliasResponse{RoomID: "!board:localhost"}, nil
	}
	var joinedRoom string
	joiner := func(roomID string) (string, error) {
		joinedRoom = roomID
		return roomID, nil
	}
	err := runJoin("construct-board", resolver, joiner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, resolvedAlias, "#construct-board:localhost")
	jtesting.AssertEqual(t, joinedRoom, "!board:localhost")
}

func TestRunJoinFullAlias(t *testing.T) {
	var resolvedAlias string
	resolver := func(alias string) (*AliasResponse, error) {
		resolvedAlias = alias
		return &AliasResponse{RoomID: "!abc:example.com"}, nil
	}
	joiner := func(roomID string) (string, error) {
		return roomID, nil
	}
	err := runJoin("#general:example.com", resolver, joiner)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, resolvedAlias, "#general:example.com")
}
