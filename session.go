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

func buildShellCmd(team string, profile Profile, _, token, ghToken string) string {
	var parts []string

	// Set git identity.
	if profile.Git.Name != "" {
		parts = append(parts, fmt.Sprintf("git config user.name %q", profile.Git.Name))
	}
	if profile.Git.Email != "" {
		parts = append(parts, fmt.Sprintf("git config user.email %q", profile.Git.Email))
	}

	if team != "" {
		parts = append(parts, fmt.Sprintf("export JACK_TEAM=%s", team))
	}
	if token != "" {
		parts = append(parts, fmt.Sprintf("export JACK_MSG_TOKEN=%s", token))
	}
	if ghToken != "" {
		parts = append(parts, fmt.Sprintf("export GH_TOKEN=%s", ghToken))
	}

	parts = append(parts, "exec claude --dangerously-skip-permissions")
	return strings.Join(parts, " && ")
}

// buildEnvFile creates the content for a .jack/env file containing session
// environment variables. Commands spawned by Claude read this file as a
// fallback when env vars are not inherited from the process environment.
func buildEnvFile(team, token, ghToken string) string {
	var lines []string
	if team != "" {
		lines = append(lines, "JACK_TEAM="+team)
	}
	if token != "" {
		lines = append(lines, "JACK_MSG_TOKEN="+token)
	}
	if ghToken != "" {
		lines = append(lines, "GH_TOKEN="+ghToken)
	}
	return strings.Join(lines, "\n") + "\n"
}

// buildSandboxShellCmd builds a shell command that launches Claude inside a
// Linux namespace sandbox. Kept for future use once credential forwarding is
// resolved.
//
//nolint:unused // intentionally kept for future use
func buildSandboxShellCmd(team string, profile Profile, dir, token, ghToken string) string {
	sock := os.Getenv("SSH_AUTH_SOCK")

	var parts []string

	if profile.Git.Name != "" {
		parts = append(parts, fmt.Sprintf("git config user.name %q", profile.Git.Name))
	}
	if profile.Git.Email != "" {
		parts = append(parts, fmt.Sprintf("git config user.email %q", profile.Git.Email))
	}

	s := make([]string, 0, 20)
	s = append(s, "set -e")
	s = append(s, "root=$(mktemp -d)", "mount -t tmpfs tmpfs $root")

	for _, d := range []string{"usr", "lib", "lib64", "bin", "sbin", "etc"} {
		s = append(s, fmt.Sprintf(
			"[ -d /%s ] && mkdir -p $root/%s && mount --bind /%s $root/%s && mount -o remount,ro,bind $root/%s",
			d, d, d, d, d,
		))
	}

	s = append(s,
		"mkdir -p $root/dev $root/proc $root/tmp",
		"mount --rbind /dev $root/dev",
		"mount -t proc proc $root/proc",
		"mount -t tmpfs tmpfs $root/tmp",
	)

	home, _ := os.UserHomeDir()
	s = append(s, "mkdir -p $root/home/project")
	s = append(s, fmt.Sprintf("mount --bind %s $root/home/project", dir))

	claudeLocal := filepath.Join(home, ".claude", "local")
	if _, err := os.Stat(claudeLocal); err == nil {
		s = append(s,
			fmt.Sprintf("mkdir -p $root%s", claudeLocal),
			fmt.Sprintf("mount --bind %s $root%s", claudeLocal, claudeLocal),
		)
	}

	claudeHome := filepath.Join(home, ".claude")
	credFile := filepath.Join(claudeHome, ".credentials.json")
	if _, err := os.Stat(credFile); err == nil {
		s = append(s,
			"mkdir -p $root/claude-config",
			"touch $root/claude-config/.credentials.json",
			fmt.Sprintf("mount --bind %s $root/claude-config/.credentials.json", credFile),
		)
	}

	if sock != "" {
		sockDir := filepath.Dir(sock)
		s = append(s,
			fmt.Sprintf("mkdir -p $root%s", sockDir),
			fmt.Sprintf("mount --bind %s $root%s", sockDir, sockDir),
		)
	}

	s = append(s,
		"pivot_root $root $root/tmp",
		"umount -l /tmp",
		"mount -t tmpfs tmpfs /tmp",
	)

	s = append(s, "cd /home/project", "export HOME=/home/project")
	s = append(s, "export CLAUDE_CONFIG_DIR=/claude-config")
	if sock != "" {
		s = append(s, fmt.Sprintf("export SSH_AUTH_SOCK=%s", sock))
	}
	if team != "" {
		s = append(s, fmt.Sprintf("export JACK_TEAM=%s", team))
	}
	if token != "" {
		s = append(s, fmt.Sprintf("export JACK_MSG_TOKEN=%s", token))
	}
	if ghToken != "" {
		s = append(s, fmt.Sprintf("export GH_TOKEN=%s", ghToken))
	}
	s = append(s, "exec unshare --user --map-user=1000 --map-group=1000 --fork -- claude --dangerously-skip-permissions")

	script := strings.Join(s, "; ")
	sandbox := fmt.Sprintf("exec unshare --mount --user --map-root-user --pid --fork sh -c '%s'", script)

	parts = append(parts, sandbox)
	return strings.Join(parts, " && ")
}
