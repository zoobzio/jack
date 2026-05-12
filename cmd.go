// Package jack provides a CLI for managing agentic development sessions.
package jack

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var env Env

var rootCmd = &cobra.Command{
	Use:   "jack",
	Short: "Operator console for multi-agent development",
	Long:  "Jack manages agents, sandboxes, sessions, and profiles for multi-agent Claude Code development.",
	PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
		env = loadEnv()
		if err := initConfig(env.configPath()); err != nil {
			return err
		}
		return nil
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
