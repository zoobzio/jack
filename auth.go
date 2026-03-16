package jack

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	authCmd.Flags().StringP("team", "t", "", "team name (required)")
	_ = authCmd.MarkFlagRequired("team")
	rootCmd.AddCommand(authCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Store a GitHub token for a team",
	Long:  "Encrypt and store a GitHub personal access token for a team profile.\nThe token is encrypted with the team's SSH public key and used to set GH_TOKEN in sessions.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		team, _ := cmd.Flags().GetString("team")
		return runAuth(team, ageEncrypt)
	},
}

func runAuth(team string, encrypt TokenEncrypter) error {
	profile, ok := cfg.Profiles[team]
	if !ok {
		return fmt.Errorf("unknown team %q (no matching profile)", team)
	}

	if profile.SSH.Key == "" {
		return fmt.Errorf("team %q has no SSH key configured", team)
	}

	fmt.Printf("Enter GitHub personal access token for %s (%s): ", team, profile.GitHub.User)
	reader := bufio.NewReader(os.Stdin)
	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("reading token: %w", err)
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token must not be empty")
	}

	pubKeyPath := expandHome(profile.SSH.Key) + ".pub"
	outPath := ghTokenAgePath(team)

	if err := encrypt(token, pubKeyPath, outPath); err != nil {
		return fmt.Errorf("encrypting github token: %w", err)
	}

	fmt.Printf("GitHub token stored for team %s at %s\n", team, outPath)
	return nil
}
