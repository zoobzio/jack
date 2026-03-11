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
	runCmd.Flags().StringP("team", "t", "", "team to use for the session (required)")
	_ = runCmd.MarkFlagRequired("team")
	runCmd.Flags().Bool("detach", false, "create session in the background without attaching")
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a session",
	Long:  "Create a tmux session running Claude Code inside a bubblewrap sandbox.\nUses the current directory name as the session name.\nAttaches to the session by default; use --detach to run in the background.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		team, _ := cmd.Flags().GetString("team")
		detach, _ := cmd.Flags().GetBool("detach")
		dir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}
		repo := filepath.Base(dir)
		return runRun(repo, team, dir, detach, HasSession, CreateSession, AttachSession, sshAdd, ageDecrypt)
	},
}

func runRun(repo, teamName, dir string, detach bool, hasSession SessionChecker, createSession SessionCreator, attach SessionAttacher, addKey KeyAdder, decrypt TokenDecrypter) error {
	profile, ok := cfg.Profiles[teamName]
	if !ok {
		return fmt.Errorf("unknown team %q (no matching profile)", teamName)
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

	// Decrypt session token if present.
	var token string
	agePath := tokenAgePath(dir)
	if _, err := os.Stat(agePath); err == nil {
		privKeyPath := expandHome(profile.SSH.Key)
		t, err := decrypt(privKeyPath, agePath)
		if err != nil {
			return fmt.Errorf("decrypting token: %w", err)
		}
		token = t
	}

	shellCmd := buildShellCmd(profile, dir, token)
	if err := createSession(name, dir, shellCmd); err != nil {
		return err
	}

	if detach {
		fmt.Printf("created session %s\n", name)
		return nil
	}

	return attach(name)
}

func sshAdd(key string) error {
	cmd := exec.CommandContext(context.Background(), "ssh-add", key)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildShellCmd(profile Profile, dir, token string) string {
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
	if token != "" {
		bwrap = append(bwrap, fmt.Sprintf("--setenv JACK_MSG_TOKEN %s", token))
	}
	bwrap = append(bwrap, "-- claude --dangerously-skip-permissions")

	parts = append(parts, strings.Join(bwrap, " "))
	return strings.Join(parts, " && ")
}
