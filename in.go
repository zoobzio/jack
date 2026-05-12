package jack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	inCmd.Flags().StringP("agent", "a", "", "agent name")
	inCmd.Flags().StringP("project", "p", "", "project name")
	inCmd.Flags().StringP("worktree", "w", "", "branch name for worktree")
	rootCmd.AddCommand(inCmd)
}

var inCmd = &cobra.Command{
	Use:   "in",
	Short: "Enter a session",
	Long:  "Attach to an existing session or create one.\nWith no arguments, interactively select an agent and project.\nUse -w to create or attach to a worktree for parallel work.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		agent, _ := cmd.Flags().GetString("agent")
		project, _ := cmd.Flags().GetString("project")
		branch, _ := cmd.Flags().GetString("worktree")
		return runIn(agent, project, branch,
			loadRegistry,
			selectAgent, selectProject,
			HasSession, CreateSession, AttachSession,
			DockerRun, DockerExec, DockerStop, DockerCheck,
		)
	},
}

func runIn(agent, project, branch string, loadReg RegistryLoader, selAgent AgentSelector, selProject ProjectSelector, hasSession SessionChecker, createSession SessionCreator, attach SessionAttacher, runContainer ContainerRunner, execContainer ContainerExecer, stopContainer ContainerStopper, checkContainer ContainerChecker) error {
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
			a, selErr := selAgent(agents)
			if selErr != nil {
				return selErr
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
			p, selErr := selProject(agent, repos)
			if selErr != nil {
				return selErr
			}
			project = p
		}
	}

	// Verify agent exists in config.
	profile, ok := cfg.Profiles[agent]
	if !ok {
		return fmt.Errorf("unknown agent %q (no matching profile)", agent)
	}

	// Check if -w matches the main clone's branch — if so, treat as main session.
	if branch != "" {
		dir := filepath.Join(env.dataDir(), agent, project)
		if mainBr := readHEADBranch(dir); mainBr != "" && mainBr == branch {
			branch = ""
		}
	}

	name := SessionName(agent, project, branch)

	// If session exists, attach to it.
	if hasSession(name) {
		return attach(name)
	}

	dir := filepath.Join(env.dataDir(), agent, project)
	containerName := ContainerName(agent, project)

	// Renew agent certificate if expiring soon.
	if cfg.CA.URL != "" && certNeedsRenewal(agent, renewThreshold) {
		if renewErr := renewCert(context.Background(), agent); renewErr != nil {
			fmt.Fprintf(os.Stderr, "warning: cert renewal failed for %s: %v\n", agent, renewErr)
		}
	}

	// Sync Claude OAuth credentials from keychain to disk.
	if err := syncClaudeCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not sync claude credentials: %v\n", err)
	}

	// Ensure the container is running (may already be up for another worktree).
	running, _ := checkContainer(containerName)
	if !running {
		mounts := SessionMounts(profile, agent, dir)
		mounts = append(mounts, Mount{
			Source:   env.configDir(),
			Target:  "/home/jack/.config/jack",
			ReadOnly: true,
		})
		volumes := []Volume{ToolsVolume(agent, project)}
		envVars := SessionEnv(profile, agent)

		if err := runContainer(containerName, mounts, volumes, envVars); err != nil {
			return fmt.Errorf("starting container: %w", err)
		}

		// Run setup scripts on fresh container start.
		scripts := setupScripts(agent, project)
		for _, s := range scripts {
			if _, err := os.Stat(s.hostPath); err != nil {
				continue
			}
			fmt.Printf("running %s...\n", s.label)
			if err := execContainer(containerName, []string{"sh", s.containerPath}); err != nil {
				_ = stopContainer(containerName)
				return fmt.Errorf("running %s: %w", s.label, err)
			}
		}
	}

	// If worktree requested, create it inside the container if it doesn't exist.
	workdir := "/home/jack/workspace/repo"
	if branch != "" {
		wtDir := WorktreeContainerPath(project, branch)
		// Create the worktree if it doesn't exist.
		if err := execContainer(containerName, []string{
			"sh", "-c",
			fmt.Sprintf("test -d %s || git -C /home/jack/workspace/repo worktree add %s %s",
				wtDir, wtDir, branch),
		}); err != nil {
			return fmt.Errorf("creating worktree for branch %s: %w", branch, err)
		}
		workdir = wtDir
	}

	// Build the tmux command as docker exec into the container.
	shellCmd := fmt.Sprintf("cd %s && exec claude --dangerously-skip-permissions", workdir)
	tmuxCmd := DockerExecCmd(containerName, shellCmd)

	if err := createSession(name, dir, tmuxCmd); err != nil {
		if !running {
			_ = stopContainer(containerName)
		}
		return err
	}

	return attach(name)
}

// readHEADBranch reads the branch name from a repo's .git/HEAD on the host.
func readHEADBranch(repoDir string) string {
	data, err := os.ReadFile(filepath.Join(repoDir, ".git", "HEAD")) // #nosec G304 -- path from internal data dir
	if err != nil {
		return ""
	}
	head := strings.TrimSpace(string(data))
	const prefix = "ref: refs/heads/"
	if strings.HasPrefix(head, prefix) {
		return strings.TrimPrefix(head, prefix)
	}
	return ""
}

type setupScript struct {
	hostPath      string
	containerPath string
	label         string
}

// setupScripts returns the ordered list of setup scripts to run on jack in.
func setupScripts(agent, project string) []setupScript {
	configDir := env.configDir()
	const containerConfig = "/home/jack/.config/jack"
	return []setupScript{
		{
			hostPath:      filepath.Join(configDir, "setup.sh"),
			containerPath: containerConfig + "/setup.sh",
			label:         "global setup",
		},
		{
			hostPath:      filepath.Join(configDir, "agents", agent, "setup.sh"),
			containerPath: containerConfig + "/agents/" + agent + "/setup.sh",
			label:         "agent setup for " + agent,
		},
		{
			hostPath:      filepath.Join(configDir, "projects", project, "dev.sh"),
			containerPath: containerConfig + "/projects/" + project + "/dev.sh",
			label:         "project setup for " + project,
		},
	}
}
