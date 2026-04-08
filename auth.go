package jack

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	authCmd.Flags().StringP("agent", "a", "", "agent name (required)")
	_ = authCmd.MarkFlagRequired("agent")
	rootCmd.AddCommand(authCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Store a GitHub token for an agent",
	Long:  "Encrypt and store a GitHub personal access token for an agent profile.\nThe token is encrypted with the agent's SSH public key and used to set GH_TOKEN in sessions.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		agent, _ := cmd.Flags().GetString("agent")
		return runAuth(agent, ageEncrypt)
	},
}

func runAuth(agent string, encrypt TokenEncrypter) error {
	profile, ok := cfg.Profiles[agent]
	if !ok {
		return fmt.Errorf("unknown agent %q (no matching profile)", agent)
	}

	if profile.SSH.Key == "" {
		return fmt.Errorf("agent %q has no SSH key configured", agent)
	}

	fmt.Printf("Enter GitHub personal access token for %s (%s): ", agent, profile.GitHub.User)
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
	outPath := ghTokenAgePath(agent)

	if err := encrypt(token, pubKeyPath, outPath); err != nil {
		return fmt.Errorf("encrypting github token: %w", err)
	}

	fmt.Printf("GitHub token stored for agent %s at %s\n", agent, outPath)
	return nil
}
