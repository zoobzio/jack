package jack

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	pruneCmd.Flags().StringP("agent", "a", "", "agent name (required)")
	pruneCmd.Flags().StringP("project", "p", "", "project name (required)")
	pruneCmd.Flags().Bool("force", false, "skip confirmation")
	pruneCmd.Flags().Bool("dry-run", false, "list what would be pruned without acting")
	_ = pruneCmd.MarkFlagRequired("agent")
	_ = pruneCmd.MarkFlagRequired("project")
	rootCmd.AddCommand(pruneCmd)
}

var pruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove unused worktrees",
	Long:  "Remove worktrees that have no active tmux session.\nThe main clone is never pruned.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		agent, _ := cmd.Flags().GetString("agent")
		project, _ := cmd.Flags().GetString("project")
		force, _ := cmd.Flags().GetBool("force")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		return runPrune(agent, project, force, dryRun, HasSession, DockerCheck, DockerExec)
	},
}

func runPrune(agent, project string, force, dryRun bool, hasSession SessionChecker, checkContainer ContainerChecker, execContainer ContainerExecer) error {
	worktrees := listWorktreeBranches(agent, project)
	if len(worktrees) == 0 {
		fmt.Println("no worktrees found")
		return nil
	}

	containerName := ContainerName(agent, project)
	running, _ := checkContainer(containerName)

	// Find worktrees with no active session.
	var prunable []string
	for branch, hash := range worktrees {
		name := SessionName(agent, project, branch)
		if !hasSession(name) {
			prunable = append(prunable, branch)
			_ = hash // used indirectly via SessionName
		}
	}

	if len(prunable) == 0 {
		fmt.Println("no unused worktrees to prune")
		return nil
	}

	// Display what would be pruned.
	for _, branch := range prunable {
		fmt.Printf("  %s\n", branch)
	}

	if dryRun {
		fmt.Printf("%d worktree(s) would be pruned\n", len(prunable))
		return nil
	}

	// Confirm unless --force.
	if !force {
		fmt.Printf("prune %d worktree(s)? [y/N] ", len(prunable))
		reader := bufio.NewReader(os.Stdin)
		answer, _ := reader.ReadString('\n')
		if strings.TrimSpace(strings.ToLower(answer)) != "y" {
			fmt.Println("cancelled")
			return nil
		}
	}

	if !running {
		return fmt.Errorf("container %s is not running — start it with jack in first", containerName)
	}

	var removed int
	for _, branch := range prunable {
		wtDir := WorktreeContainerPath(project, branch)
		if err := execContainer(containerName, []string{
			"git", "-C", "/home/jack/workspace/repo", "worktree", "remove", wtDir,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not remove worktree %s: %v\n", branch, err)
			continue
		}
		fmt.Printf("removed %s\n", branch)
		removed++
	}

	fmt.Printf("pruned %d worktree(s)\n", removed)
	return nil
}
