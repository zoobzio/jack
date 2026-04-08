//go:build testing

package msg

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunWhoSuccess(t *testing.T) {
	dir := t.TempDir()
	regContent := `projects:
  - agent: blue
    repo: vicky
  - agent: blue
    repo: flux
`
	_ = os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(regContent), 0o600)

	Homeserver = "http://localhost:8008"
	err := runWho(dir)
	jtesting.AssertNoError(t, err)
}

func TestRunWhoEmptyDataDir(t *testing.T) {
	err := runWho("")
	jtesting.AssertError(t, err)
}

func TestRunWhoMissingRegistry(t *testing.T) {
	err := runWho(t.TempDir())
	jtesting.AssertNoError(t, err)
}

func TestRunWhoOnlineSuccess(t *testing.T) {
	dir := t.TempDir()
	regContent := `projects:
  - agent: blue
    repo: vicky
`
	_ = os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(regContent), 0o600)
	Homeserver = "http://localhost:8008"

	getter := func(userID string) (*PresenceResponse, error) {
		return &PresenceResponse{
			Presence:        "online",
			CurrentlyActive: true,
		}, nil
	}
	resolver := func(_ string) (*AliasResponse, error) {
		return &AliasResponse{RoomID: "!board:localhost"}, nil
	}
	members := func(roomID string) ([]Member, error) {
		return []Member{
			{UserID: "@blue-vicky:localhost", DisplayName: "blue-vicky"},
		}, nil
	}
	err := runWhoOnline(dir, getter, resolver, members)
	jtesting.AssertNoError(t, err)
}

func TestRunWhoOnlineOffline(t *testing.T) {
	dir := t.TempDir()
	regContent := `projects:
  - agent: blue
    repo: vicky
`
	_ = os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(regContent), 0o600)
	Homeserver = "http://localhost:8008"

	getter := func(userID string) (*PresenceResponse, error) {
		return &PresenceResponse{
			Presence:      "offline",
			LastActiveAgo: 300000, // 5 minutes
		}, nil
	}
	resolver := func(_ string) (*AliasResponse, error) {
		return nil, fmt.Errorf("not found")
	}
	members := func(_ string) ([]Member, error) {
		return nil, fmt.Errorf("not found")
	}
	err := runWhoOnline(dir, getter, resolver, members)
	jtesting.AssertNoError(t, err)
}

func TestRunWhoOnlineError(t *testing.T) {
	dir := t.TempDir()
	regContent := `projects:
  - agent: blue
    repo: vicky
`
	_ = os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(regContent), 0o600)
	Homeserver = "http://localhost:8008"

	getter := func(userID string) (*PresenceResponse, error) {
		return nil, fmt.Errorf("presence disabled")
	}
	resolver := func(_ string) (*AliasResponse, error) {
		return nil, fmt.Errorf("not found")
	}
	members := func(_ string) ([]Member, error) {
		return nil, fmt.Errorf("not found")
	}
	// Should not error — prints "unknown" for unreachable presence.
	err := runWhoOnline(dir, getter, resolver, members)
	jtesting.AssertNoError(t, err)
}

func TestRunWhoOnlineBoardMembership(t *testing.T) {
	dir := t.TempDir()
	regContent := `projects:
  - agent: rockhopper
    repo: sentinel
  - agent: wintermute
    repo: sentinel
`
	_ = os.WriteFile(filepath.Join(dir, "registry.yaml"), []byte(regContent), 0o600)
	Homeserver = "http://localhost:8008"

	getter := func(userID string) (*PresenceResponse, error) {
		return &PresenceResponse{Presence: "online", CurrentlyActive: true}, nil
	}
	resolver := func(alias string) (*AliasResponse, error) {
		return &AliasResponse{RoomID: "!board:localhost"}, nil
	}
	members := func(roomID string) ([]Member, error) {
		return []Member{
			{UserID: "@rockhopper-sentinel:localhost"},
		}, nil
	}
	// rockhopper-sentinel should show board: yes, wintermute-sentinel board: no
	err := runWhoOnline(dir, getter, resolver, members)
	jtesting.AssertNoError(t, err)
}
