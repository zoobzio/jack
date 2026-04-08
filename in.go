package jack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/zoobzio/jack/msg"
)

func init() {
	inCmd.Flags().StringP("agent", "a", "", "agent name")
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
	Long:  "Attach to an existing session or create one.\nWith no arguments, interactively select an agent and project.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		agent, _ := cmd.Flags().GetString("agent")
		project, _ := cmd.Flags().GetString("project")
		return runIn(agent, project,
			loadRegistry,
			selectAgent, selectProject,
			HasSession, CreateSession, AttachSession,
			sshAdd, ageDecrypt,
			defaultBoardProvisioner,
		)
	},
}

func runIn(agent, project string, loadReg RegistryLoader, selAgent AgentSelector, selProject ProjectSelector, hasSession SessionChecker, createSession SessionCreator, attach SessionAttacher, addKey KeyAdder, decrypt TokenDecrypter, provision BoardProvisioner) error {
	reg, err := loadReg()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	// Resolve agent.
	if agent == "" {
		agents := reg.Agents()
		switch len(agents) {
		case 0:
			return fmt.Errorf("no projects cloned — run jack clone first")
		case 1:
			agent = agents[0]
		default:
			a, err := selAgent(agents)
			if err != nil {
				return err
			}
			agent = a
		}
	}

	// Resolve project.
	if project == "" {
		repos := reg.ReposForAgent(agent)
		switch len(repos) {
		case 0:
			return fmt.Errorf("no projects cloned for agent %q", agent)
		case 1:
			project = repos[0]
		default:
			p, err := selProject(agent, repos)
			if err != nil {
				return err
			}
			project = p
		}
	}

	name := SessionName(agent, project)
	dir := filepath.Join(env.dataDir(), agent, project)

	// If session exists, attach to it.
	if hasSession(name) {
		return attach(name)
	}

	// Create a new session.
	profile, ok := cfg.Profiles[agent]
	if !ok {
		return fmt.Errorf("unknown agent %q (no matching profile)", agent)
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
	ghAgePath := ghTokenAgePath(agent)
	if _, err := os.Stat(ghAgePath); err == nil {
		privKeyPath := expandHome(profile.SSH.Key)
		t, err := decrypt(privKeyPath, ghAgePath)
		if err != nil {
			return fmt.Errorf("decrypting github token: %w", err)
		}
		ghToken = t
	}

	// Provision global board and announce presence (non-fatal).
	if token != "" && provision != nil && boardAutoJoinMatch(agent) {
		if err := provision(token, name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: global board provisioning failed: %v\n", err)
		}
	}

	shellCmd := buildShellCmd(agent, profile, dir, token, ghToken)

	// Write session env vars to a file so that commands spawned by Claude
	// (which starts a fresh shell from the user's profile rather than
	// inheriting the process environment) can still read them.
	jackDir := filepath.Join(dir, ".jack")
	_ = os.MkdirAll(jackDir, 0o750)
	envContent := buildEnvFile(agent, token, ghToken)
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

// boardAutoJoinMatch reports whether the agent name matches the configured
// board_auto_join regex pattern. An empty pattern matches all agents.
func boardAutoJoinMatch(agent string) bool {
	pattern := cfg.Matrix.BoardAutoJoin
	if pattern == "" {
		return true
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: invalid board_auto_join pattern %q: %v\n", pattern, err)
		return false
	}
	return re.MatchString(agent)
}
