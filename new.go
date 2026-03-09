package jack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// SessionChecker reports whether a tmux session exists.
type SessionChecker func(string) bool

// SessionCreator creates a detached tmux session.
type SessionCreator func(name, dir, shellCmd string) error

// KeyAdder adds an SSH key to the agent.
type KeyAdder func(key string) error

func init() {
	newCmd.Flags().StringP("team", "t", "", "team to use for the session (required)")
	_ = newCmd.MarkFlagRequired("team")
	rootCmd.AddCommand(newCmd)
}

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new session",
	Long:  "Create a new tmux session running Claude Code inside a bubblewrap sandbox.\nUses the current directory name as the session name.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		team, _ := cmd.Flags().GetString("team")
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		repo := filepath.Base(dir)
		return runNew(repo, team, dir, HasSession, CreateSession, sshAdd)
	},
}

func runNew(repo, teamName, dir string, hasSession SessionChecker, createSession SessionCreator, addKey KeyAdder) error {
	team, ok := cfg.Teams[teamName]
	if !ok {
		return fmt.Errorf("unknown team %q", teamName)
	}

	profile, ok := cfg.Profiles[team.Profile]
	if !ok {
		return fmt.Errorf("team %q references unknown profile %q", teamName, team.Profile)
	}

	name := SessionName(teamName, repo)
	if hasSession(name) {
		return fmt.Errorf("session %q already exists", name)
	}

	// Add the profile's SSH key to the agent.
	if profile.SSH.Key != "" {
		key := expandHome(profile.SSH.Key)
		if err := addKey(key); err != nil {
			return fmt.Errorf("ssh-add %s: %w", key, err)
		}
	}

	shellCmd := buildShellCmd(profile, dir)
	if err := createSession(name, dir, shellCmd); err != nil {
		return err
	}

	fmt.Printf("created session %s\n", name)
	return nil
}

func sshAdd(key string) error {
	cmd := exec.CommandContext(context.Background(), "ssh-add", key)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildShellCmd(profile Profile, dir string) string {
	home, _ := os.UserHomeDir()
	sock := os.Getenv("SSH_AUTH_SOCK")

	var parts []string

	// Set git identity.
	if profile.Git.Name != "" {
		parts = append(parts, fmt.Sprintf("git config user.name %q", profile.Git.Name))
	}
	if profile.Git.Email != "" {
		parts = append(parts, fmt.Sprintf("git config user.email %q", profile.Git.Email))
	}

	// Build bwrap command.
	bwrap := []string{
		"exec bwrap",
		"--ro-bind / /",
		"--dev /dev",
		"--proc /proc",
		"--tmpfs /tmp",
		fmt.Sprintf("--bind %s %s", dir, dir),
		"--ro-bind-try ~/.claude ~/.claude",
		"--ro-bind-try ~/.config/claude ~/.config/claude",
		fmt.Sprintf("--setenv HOME %s", home),
	}
	if sock != "" {
		bwrap = append(bwrap, fmt.Sprintf("--setenv SSH_AUTH_SOCK %s", sock))
	}
	bwrap = append(bwrap, "-- claude --dangerously-skip-permissions")

	parts = append(parts, strings.Join(bwrap, " "))
	return strings.Join(parts, " && ")
}
