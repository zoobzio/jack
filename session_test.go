//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

// Shared test helpers used across multiple test files.

func newTestConfig() {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {
				Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"},
				SSH: SSHConfig{Key: "~/.ssh/id_rock"},
			},
		},
	}
}

func noopChecker(string) bool                  { return false }
func existsChecker(string) bool                { return true }
func noopCreator(_, _, _ string) error          { return nil }
func noopAdder(_ string) error                  { return nil }
func noopDecrypter(_, _ string) (string, error) { return "", nil }
func noopAttacher(_ string) error               { return nil }

func TestBuildShellCmd(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd("blue", profile, "/home/user/project", "", "")

	jtesting.AssertEqual(t, strings.Contains(cmd, `git config user.name "Test User"`), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, `git config user.email "test@example.com"`), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "claude --dangerously-skip-permissions --teammate-mode in-process"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, " && "), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "export JACK_AGENT=blue"), true)
}

func TestBuildShellCmdNoGitConfig(t *testing.T) {
	profile := Profile{}

	cmd := buildShellCmd("", profile, "/tmp", "", "")

	jtesting.AssertEqual(t, strings.Contains(cmd, "git config"), false)
	jtesting.AssertEqual(t, cmd, "exec claude --dangerously-skip-permissions --teammate-mode in-process")
}

func TestBuildShellCmdWithToken(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd("blue", profile, "/home/user/project", "tok_session", "")
	jtesting.AssertEqual(t, strings.Contains(cmd, "export JACK_MSG_TOKEN=tok_session"), true)
}

func TestBuildShellCmdWithGHToken(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd("blue", profile, "/home/user/project", "", "ghp_abc123")
	jtesting.AssertEqual(t, strings.Contains(cmd, "export GH_TOKEN=ghp_abc123"), true)
}

func TestBuildShellCmdWithBothTokens(t *testing.T) {
	profile := Profile{
		Git: GitConfig{Name: "Test User", Email: "test@example.com"},
	}

	cmd := buildShellCmd("blue", profile, "/home/user/project", "tok_session", "ghp_abc123")
	jtesting.AssertEqual(t, strings.Contains(cmd, "export JACK_MSG_TOKEN=tok_session"), true)
	jtesting.AssertEqual(t, strings.Contains(cmd, "export GH_TOKEN=ghp_abc123"), true)
}

func TestBuildEnvFile(t *testing.T) {
	content := buildEnvFile("blue", "tok_123", "ghp_abc")
	jtesting.AssertEqual(t, strings.Contains(content, "JACK_AGENT=blue\n"), true)
	jtesting.AssertEqual(t, strings.Contains(content, "JACK_MSG_TOKEN=tok_123\n"), true)
	jtesting.AssertEqual(t, strings.Contains(content, "GH_TOKEN=ghp_abc\n"), true)
}

func TestBuildEnvFileEmpty(t *testing.T) {
	content := buildEnvFile("", "", "")
	jtesting.AssertEqual(t, content, "\n")
}
