package jack

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zoobzio/jack/msg"
)

func init() {
	inCmd.Flags().StringP("team", "t", "", "team name")
	inCmd.Flags().StringP("project", "p", "", "project name")
	rootCmd.AddCommand(inCmd)
}

// BoardProvisioner joins the global board and announces presence.
type BoardProvisioner func(token, sessionName string) error

func defaultBoardProvisioner(token, sessionName string) error {
	if err := msg.ProvisionGlobalBoard(token); err != nil {
		return err
	}
	return msg.AnnounceOnBoard(token, fmt.Sprintf("%s jacked in", sessionName))
}

var inCmd = &cobra.Command{
	Use:   "in",
	Short: "Enter a session",
	Long:  "Attach to an existing session or create one.\nWith no arguments, interactively select a team and project.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		team, _ := cmd.Flags().GetString("team")
		project, _ := cmd.Flags().GetString("project")
		return runIn(team, project,
			loadRegistry,
			selectTeam, selectProject,
			HasSession, CreateSession, AttachSession,
			sshAdd, ageDecrypt,
			defaultBoardProvisioner,
		)
	},
}

func runIn(team, project string, loadReg RegistryLoader, selTeam TeamSelector, selProject ProjectSelector, hasSession SessionChecker, createSession SessionCreator, attach SessionAttacher, addKey KeyAdder, decrypt TokenDecrypter, provision BoardProvisioner) error {
	reg, err := loadReg()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	// Resolve team.
	if team == "" {
		teams := reg.Teams()
		switch len(teams) {
		case 0:
			return fmt.Errorf("no projects cloned — run jack clone first")
		case 1:
			team = teams[0]
		default:
			t, err := selTeam(teams)
			if err != nil {
				return err
			}
			team = t
		}
	}

	// Resolve project.
	if project == "" {
		repos := reg.ReposForTeam(team)
		switch len(repos) {
		case 0:
			return fmt.Errorf("no projects cloned for team %q", team)
		case 1:
			project = repos[0]
		default:
			p, err := selProject(team, repos)
			if err != nil {
				return err
			}
			project = p
		}
	}

	name := SessionName(team, project)
	dir := filepath.Join(env.dataDir(), team, project)

	// If session exists, attach to it.
	if hasSession(name) {
		return attach(name)
	}

	// Create a new session.
	profile, ok := cfg.Profiles[team]
	if !ok {
		return fmt.Errorf("unknown team %q (no matching profile)", team)
	}

	if profile.SSH.Key != "" {
		key := expandHome(profile.SSH.Key)
		if err := addKey(key); err != nil {
			return fmt.Errorf("ssh-add %s: %w", key, err)
		}
	}

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

	var ghToken string
	ghAgePath := ghTokenAgePath(team)
	if _, err := os.Stat(ghAgePath); err == nil {
		privKeyPath := expandHome(profile.SSH.Key)
		t, err := decrypt(privKeyPath, ghAgePath)
		if err != nil {
			return fmt.Errorf("decrypting github token: %w", err)
		}
		ghToken = t
	}

	// Provision global board and announce presence (non-fatal).
	if token != "" && provision != nil {
		if err := provision(token, name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: global board provisioning failed: %v\n", err)
		}
	}

	shellCmd := buildShellCmd(team, profile, dir, token, ghToken)

	// Write session env vars to a file so that commands spawned by Claude
	// (which starts a fresh shell from the user's profile rather than
	// inheriting the process environment) can still read them.
	jackDir := filepath.Join(dir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	envContent := buildEnvFile(team, token, ghToken)
	if err := os.WriteFile(filepath.Join(jackDir, "env"), []byte(envContent), 0o600); err != nil {
		return fmt.Errorf("writing env file: %w", err)
	}

	// Write to a script file so tmux doesn't have to handle long inline
	// commands. Capture stderr to a log file for diagnostics.
	scriptPath := filepath.Join(jackDir, "session.sh")
	logPath := filepath.Join(jackDir, "session.log")
	content := fmt.Sprintf("#!/bin/sh\n%s 2>%s\n", shellCmd, logPath)
	if err := os.WriteFile(scriptPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("writing session script: %w", err)
	}

	if err := createSession(name, dir, "sh "+scriptPath); err != nil {
		return err
	}

	return attach(name)
}
