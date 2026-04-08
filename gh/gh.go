// Package gh wraps the GitHub CLI with content classification via an external
// classifier service (ICE). All reads tag untrusted user-authored fields and
// all writes are scanned before reaching GitHub.
package gh

import (
	"github.com/spf13/cobra"
)

// Package-level config set by the parent package during PersistentPreRunE.
var ClassifierEndpoint string

// Cmd is the parent command for all gh subcommands.
var Cmd = &cobra.Command{
	Use:   "gh",
	Short: "Classified GitHub CLI",
	Long:  "GitHub operations with content classification. Reads tag untrusted fields, writes are scanned before submission.",
}
