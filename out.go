package jack

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	outCmd.Flags().StringP("agent", "a", "", "agent name")
	outCmd.Flags().StringP("project", "p", "", "project name")
	outCmd.Flags().StringP("worktree", "w", "", "branch name for worktree")
	outCmd.Flags().Bool("all", false, "terminate all sessions")
	rootCmd.AddCommand(outCmd)
}

// parseSessionName splits a session name into agent and project.
// Agent names cannot contain hyphens, so the first hyphen is the delimiter.
func parseSessionName(name string) (agent, project string) {
	parts := strings.SplitN(name, "-", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return name, ""
}

var outCmd = &cobra.Command{
	Use:   "out [name]",
	Short: "Terminate a session",
	Long:  "Terminate a session by name, by --agent and --project flags, or --all.\nWorktree sessions are killed but the worktree is left on disk.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if all {
			return runOutAll(loadRegistry, ListSessions, HasSession, KillSession, DockerStop)
		}
		var name string
		if len(args) > 0 {
			name = args[0]
		}
		agent, _ := cmd.Flags().GetString("agent")
		project, _ := cmd.Flags().GetString("project")
		branch, _ := cmd.Flags().GetString("worktree")
		return runOut(name, agent, project, branch, HasSession, KillSession, DockerStop)
	},
}

func runOut(name, agent, project, branch string, hasSession SessionChecker, kill SessionKiller, stopContainer ContainerStopper) error {
	if name == "" && agent != "" && project != "" {
		name = SessionName(agent, project, branch)
	}
	if name == "" {
		return fmt.Errorf("specify a session name or both --agent and --project")
	}
	if !hasSession(name) {
		return fmt.Errorf("session %q not found", name)
	}

	if err := kill(name); err != nil {
		return err
	}

	// Only stop the container if this is the main session (no worktree).
	// Worktree sessions share the container with the main session.
	if branch == "" {
		if agent == "" || project == "" {
			agent, project = parseSessionName(name)
		}
		if agent != "" && project != "" && stopContainer != nil {
			containerName := ContainerName(agent, project)
			if err := stopContainer(containerName); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not stop container %s: %v\n", containerName, err)
			}
		}
	}

	fmt.Printf("killed session %s\n", name)
	return nil
}

func runOutAll(loadReg RegistryLoader, list Lister, hasSession SessionChecker, kill SessionKiller, stopContainer ContainerStopper) error {
	reg, err := loadReg()
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	sessions, err := list()
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}

	// Build a set of jack-managed session names from the registry.
	managed := make(map[string]bool)
	for _, agent := range reg.Agents() {
		for _, entry := range reg.ForAgent(agent) {
			managed[SessionName(agent, entry.Repo, "")] = true
		}
	}

	var killed int
	for _, s := range sessions {
		if !managed[s.Name] {
			continue
		}
		if err := kill(s.Name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not kill %s: %v\n", s.Name, err)
			continue
		}
		agent, project := parseSessionName(s.Name)
		if agent != "" && project != "" && stopContainer != nil {
			containerName := ContainerName(agent, project)
			if err := stopContainer(containerName); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not stop container %s: %v\n", containerName, err)
			}
		}
		fmt.Printf("killed session %s\n", s.Name)
		killed++
	}

	fmt.Printf("killed %d session(s)\n", killed)
	return nil
}
