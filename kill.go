package jack

import (
	"fmt"

	"github.com/spf13/cobra"
)

// SessionKiller terminates a tmux session.
type SessionKiller func(name string) error

func init() {
	rootCmd.AddCommand(killCmd)
}

var killCmd = &cobra.Command{
	Use:   "kill <name>",
	Short: "Terminate a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runKill(args[0], HasSession, KillSession)
	},
}

func runKill(name string, hasSession SessionChecker, kill SessionKiller) error {
	if !hasSession(name) {
		return fmt.Errorf("session %q not found", name)
	}
	if err := kill(name); err != nil {
		return err
	}
	fmt.Printf("killed session %s\n", name)
	return nil
}
