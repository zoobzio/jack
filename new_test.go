//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func newTestConfig() {
	cfg = Config{
		Profiles: map[string]Profile{
			"rockhopper": {
				Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"},
				SSH: SSHConfig{Key: "~/.ssh/id_rock"},
			},
		},
		Roles: map[string]Role{
			"developer": {Skills: []string{"commit", "pr"}},
		},
		Teams: map[string]Team{
			"blue": {Profile: "rockhopper", Agents: []string{"zidgel"}},
		},
	}
}

func noopChecker(string) bool                    { return false }
func existsChecker(string) bool                  { return true }
func noopCreator(_, _, _ string) error            { return nil }
func noopAdder(_ string) error                    { return nil }

func TestRunNewUnknownTeam(t *testing.T) {
	newTestConfig()
	err := runNew("vicky", "bogus", "/tmp", noopChecker, noopCreator, noopAdder)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown team"), true)
}

func TestRunNewSessionExists(t *testing.T) {
	newTestConfig()
	err := runNew("vicky", "blue", "/tmp", existsChecker, noopCreator, noopAdder)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "already exists"), true)
}

func TestRunNewSuccess(t *testing.T) {
	newTestConfig()

	var createdName, createdDir, createdCmd string
	var addedKey string

	creator := func(name, dir, shellCmd string) error {
		createdName = name
		createdDir = dir
		createdCmd = shellCmd
		return nil
	}
	adder := func(key string) error {
		addedKey = key
		return nil
	}

	err := runNew("vicky", "blue", "/home/user/vicky", noopChecker, creator, adder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, createdName, "blue-vicky")
	jtesting.AssertEqual(t, createdDir, "/home/user/vicky")
	jtesting.AssertEqual(t, strings.Contains(addedKey, "id_rock"), true)
	jtesting.AssertEqual(t, strings.Contains(createdCmd, "claude --dangerously-skip-permissions"), true)
}

func TestRunNewNoSSHKey(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"bare": {Git: GitConfig{Name: "Bare", Email: "bare@example.com"}},
		},
		Teams: map[string]Team{
			"red": {Profile: "bare"},
		},
	}

	adderCalled := false
	adder := func(_ string) error {
		adderCalled = true
		return nil
	}

	err := runNew("flux", "red", "/tmp", noopChecker, noopCreator, adder)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, adderCalled, false)
}

func TestBuildShellCmd(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd(profile, "/home/user/project")

	jtesting.AssertEqual(t, strings.Contains(cmd, `git config user.name "Test User"`), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, `git config user.email "test@example.com"`), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--ro-bind / /"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--bind /home/user/project /home/user/project"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--dev /dev"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--proc /proc"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "--tmpfs /tmp"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "-- claude --dangerously-skip-permissions"), true)
	// Parts are joined with &&.
	jtesting.AssertEqual(t, strings.Contains(cmd, " && "), true)
}

func TestBuildShellCmdNoGitConfig(t *testing.T) {
	profile := Profile{}

	cmd := buildShellCmd(profile, "/tmp")

	jtesting.AssertEqual(t, strings.Contains(cmd, "git config"), false)
	// Should start directly with bwrap.
	jtesting.AssertEqual(t, strings.HasPrefix(cmd, "exec bwrap"), true)
}
