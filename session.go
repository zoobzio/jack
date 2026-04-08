package jack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SessionChecker reports whether a tmux session exists.
type SessionChecker func(string) bool

// SessionCreator creates a detached tmux session.
type SessionCreator func(name, dir, shellCmd string) error

// SessionAttacher attaches to a tmux session.
type SessionAttacher func(name string) error

// SessionKiller terminates a tmux session.
type SessionKiller func(name string) error

// KeyAdder adds an SSH key to the agent.
type KeyAdder func(key string) error

func sshAdd(key string) error {
	cmd := exec.CommandContext(context.Background(), "ssh-add", key)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildShellCmd(agent string, profile Profile, _, token, ghToken string) string {
	var parts []string

	// Set git identity.
	if profile.Git.Name != "" {
		parts = append(parts, fmt.Sprintf("git config user.name %q", profile.Git.Name))
	}
	if profile.Git.Email != "" {
		parts = append(parts, fmt.Sprintf("git config user.email %q", profile.Git.Email))
	}

	if agent != "" {
		parts = append(parts, fmt.Sprintf("export JACK_AGENT=%s", agent))
	}
	if token != "" {
		parts = append(parts, fmt.Sprintf("export JACK_MSG_TOKEN=%s", token))
	}
	if ghToken != "" {
		parts = append(parts, fmt.Sprintf("export GH_TOKEN=%s", ghToken))
	}

	parts = append(parts, "exec claude --dangerously-skip-permissions --teammate-mode in-process")
	return strings.Join(parts, " && ")
}

// buildBwrapShellCmd builds a shell command that launches Claude inside a
// bwrap sandbox. Kept for future use once bwrap integration is debugged.
//
//nolint:unused // intentionally kept for future use
func buildBwrapShellCmd(agent string, profile Profile, dir, token, ghToken string) string {
	var parts []string

	// Set git identity before entering the sandbox.
	if profile.Git.Name != "" {
		parts = append(parts, fmt.Sprintf("git config user.name %q", profile.Git.Name))
	}
	if profile.Git.Email != "" {
		parts = append(parts, fmt.Sprintf("git config user.email %q", profile.Git.Email))
	}

	// Build bwrap command.
	home, _ := os.UserHomeDir()
	configDir := env.configDir()

	bwrap := []string{"exec bwrap"}

	// Read-only base filesystem.
	bwrap = append(bwrap, "--ro-bind / /")

	// Proper /dev, /proc, writable /tmp.
	bwrap = append(bwrap, "--dev /dev", "--proc /proc", "--tmpfs /tmp")

	// Working directory read-write.
	bwrap = append(bwrap, fmt.Sprintf("--bind %s %s", dir, dir))

	// Jack config directory read-only (symlink targets resolve here).
	bwrap = append(bwrap, fmt.Sprintf("--ro-bind %s %s", configDir, configDir))

	// Claude local state (session data, caches).
	claudeLocal := filepath.Join(home, ".claude", "local")
	if _, err := os.Stat(claudeLocal); err == nil {
		bwrap = append(bwrap, fmt.Sprintf("--bind %s %s", claudeLocal, claudeLocal))
	}

	// SSH agent socket for git operations.
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		sockDir := filepath.Dir(sock)
		bwrap = append(bwrap, fmt.Sprintf("--ro-bind %s %s", sockDir, sockDir))
	}

	// Environment variables.
	if agent != "" {
		bwrap = append(bwrap, fmt.Sprintf("--setenv JACK_AGENT %s", agent))
	}
	if token != "" {
		bwrap = append(bwrap, fmt.Sprintf("--setenv JACK_MSG_TOKEN %s", token))
	}
	if ghToken != "" {
		bwrap = append(bwrap, fmt.Sprintf("--setenv GH_TOKEN %s", ghToken))
	}

	bwrap = append(bwrap, "-- claude --dangerously-skip-permissions --teammate-mode in-process")

	parts = append(parts, strings.Join(bwrap, " "))
	return strings.Join(parts, " && ")
}

// buildEnvFile creates the content for a .jack/env file containing session
// environment variables. Commands spawned by Claude read this file as a
// fallback when env vars are not inherited from the process environment.
func buildEnvFile(agent, token, ghToken string) string {
	var lines []string
	if agent != "" {
		lines = append(lines, "JACK_AGENT="+agent)
	}
	if token != "" {
		lines = append(lines, "JACK_MSG_TOKEN="+token)
	}
	if ghToken != "" {
		lines = append(lines, "GH_TOKEN="+ghToken)
	}
	return strings.Join(lines, "\n") + "\n"
}
