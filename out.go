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
	Long:  "Terminate a session by name or by --agent and --project flags.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var name string
		if len(args) > 0 {
			name = args[0]
		}
		agent, _ := cmd.Flags().GetString("agent")
		project, _ := cmd.Flags().GetString("project")
		return runOut(name, agent, project, HasSession, KillSession, DockerStop)
	},
}

func runOut(name, agent, project string, hasSession SessionChecker, kill SessionKiller, stopContainer ContainerStopper) error {
	if name == "" && agent != "" && project != "" {
		name = SessionName(agent, project)
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

	// Stop the Docker container (non-fatal — may not exist).
	if agent == "" || project == "" {
		agent, project = parseSessionName(name)
	}
	if agent != "" && project != "" && stopContainer != nil {
		containerName := ContainerName(agent, project)
		if err := stopContainer(containerName); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not stop container %s: %v\n", containerName, err)
		}
	}

	fmt.Printf("killed session %s\n", name)
	return nil
}
