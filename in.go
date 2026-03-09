package jack

import (
	"fmt"

	"github.com/spf13/cobra"
)

// SessionAttacher attaches to a tmux session.
type SessionAttacher func(name string) error

func init() {
	rootCmd.AddCommand(inCmd)
}

var inCmd = &cobra.Command{
	Use:   "in <name>",
	Short: "Attach to a session",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runIn(args[0], HasSession, AttachSession)
	},
}

func runIn(name string, hasSession SessionChecker, attach SessionAttacher) error {
	if !hasSession(name) {
		return fmt.Errorf("session %q not found", name)
	}
	return attach(name)
}
