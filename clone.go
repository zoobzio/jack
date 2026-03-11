package jack

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoobzio/jack/msg"
)

// Cloner clones a git repository into a directory.
type Cloner func(url, dir string) error

func init() {
	cloneCmd.Flags().StringSliceP("team", "t", nil, "teams to clone for (required, repeatable)")
	_ = cloneCmd.MarkFlagRequired("team")
	rootCmd.AddCommand(cloneCmd)
}

var cloneCmd = &cobra.Command{
	Use:   "clone <url>",
	Short: "Clone a repo for a team",
	Long:  "Clone a git repo into each team's isolated workspace, apply team skills, and create sessions.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		teams, _ := cmd.Flags().GetStringSlice("team")
		client := msg.NewClient(msg.Homeserver, "")
		return runClone(args[0], teams, gitClone, copyFile, HasSession, CreateSession, sshAdd, client.Register, ageEncrypt, writeDescription, ageDecrypt)
	},
}

func runClone(url string, teams []string, clone Cloner, cp FileCopier, hasSession SessionChecker, createSession SessionCreator, addKey KeyAdder, register msg.Registerer, encrypt TokenEncrypter, writeDesc DescriptionWriter, decrypt TokenDecrypter) error {
	repo := repoName(url)
	if repo == "" {
		return fmt.Errorf("cannot extract repo name from %q", url)
	}

	configDir := env.configDir()

	for _, teamName := range teams {
		// Validate governance prerequisites per team.
		if err := validateGovernance(configDir, teamName, repo); err != nil {
			return err
		}

		profile, ok := cfg.Profiles[teamName]
		if !ok {
			return fmt.Errorf("unknown team %q (no matching profile)", teamName)
		}

		dir := filepath.Join(env.dataDir(), teamName, repo)
		parent := filepath.Dir(dir)
		if err := os.MkdirAll(parent, 0o750); err != nil {
			return fmt.Errorf("creating directory %s: %w", parent, err)
		}

		if err := clone(url, dir); err != nil {
			return fmt.Errorf("cloning %s for team %s: %w", repo, teamName, err)
		}

		if err := applyTeam(teamName, repo, dir, cp); err != nil {
			return fmt.Errorf("applying team %s: %w", teamName, err)
		}

		// Register Matrix user for this session.
		username := teamName + "-" + repo
		reg, err := register(username, username, cfg.Matrix.RegistrationToken)
		if err != nil {
			return fmt.Errorf("registering Matrix user %s: %w", username, err)
		}

		// Encrypt and store the token.
		pubKeyPath := expandHome(profile.SSH.Key) + ".pub"
		if err := encrypt(reg.AccessToken, pubKeyPath, tokenAgePath(dir)); err != nil {
			return fmt.Errorf("encrypting token for %s: %w", username, err)
		}

		// Write session description.
		desc := fmt.Sprintf("team=%s repo=%s", teamName, repo)
		if err := writeDesc(descriptionPath(dir), desc); err != nil {
			return fmt.Errorf("writing description for %s: %w", username, err)
		}

		if err := runRun(repo, teamName, dir, true, hasSession, createSession, nil, addKey, decrypt); err != nil {
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
