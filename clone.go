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

// Cloner clones a git repository into a directory.
type Cloner func(url, dir string) error

func init() {
	cloneCmd.Flags().StringSliceP("team", "t", nil, "teams to clone for (required, repeatable)")
	_ = cloneCmd.MarkFlagRequired("team")
	cloneCmd.Flags().StringP("role", "r", "", "role to apply (required)")
	_ = cloneCmd.MarkFlagRequired("role")
	rootCmd.AddCommand(cloneCmd)
}

var cloneCmd = &cobra.Command{
	Use:   "clone <url>",
	Short: "Clone a repo for a team",
	Long:  "Clone a git repo into each team's isolated workspace, apply a role, and create sessions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teams, _ := cmd.Flags().GetStringSlice("team")
		role, _ := cmd.Flags().GetString("role")
		return runClone(args[0], teams, role, gitClone, copyFile, HasSession, CreateSession, sshAdd)
	},
}

func runClone(url string, teams []string, roleName string, clone Cloner, cp FileCopier, hasSession SessionChecker, createSession SessionCreator, addKey KeyAdder) error {
	repo := repoName(url)
	if repo == "" {
		return fmt.Errorf("cannot extract repo name from %q", url)
	}

	if _, ok := cfg.Roles[roleName]; !ok {
		return fmt.Errorf("unknown role %q", roleName)
	}

	for _, teamName := range teams {
		if _, ok := cfg.Teams[teamName]; !ok {
			return fmt.Errorf("unknown team %q", teamName)
		}

		dir := filepath.Join(env.dataDir(), teamName, repo)
		parent := filepath.Dir(dir)
		if err := os.MkdirAll(parent, 0o750); err != nil {
			return fmt.Errorf("creating directory %s: %w", parent, err)
		}

		if err := clone(url, dir); err != nil {
			return fmt.Errorf("cloning %s for team %s: %w", repo, teamName, err)
		}

		if err := applyRole(roleName, teamName, dir, cp); err != nil {
			return fmt.Errorf("applying role %s for team %s: %w", roleName, teamName, err)
		}

		if err := runNew(repo, teamName, dir, hasSession, createSession, addKey); err != nil {
			return fmt.Errorf("creating session for team %s: %w", teamName, err)
		}
	}

	return nil
}

func gitClone(url, dir string) error {
	cmd := exec.CommandContext(context.Background(), "git", "clone", url, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// repoName extracts the repository name from a git URL.
// Handles both SCP-style (git@host:user/repo.git) and standard URLs.
func repoName(url string) string {
	// Strip trailing .git
	name := strings.TrimSuffix(url, ".git")

	// Handle SCP-style URLs (git@github.com:user/repo)
	if i := strings.LastIndex(name, ":"); i != -1 && !strings.Contains(name, "://") {
		name = name[i+1:]
	}

	// Take last path segment
	if i := strings.LastIndex(name, "/"); i != -1 {
		name = name[i+1:]
	}

	return name
}
