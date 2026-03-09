package jack

import (
	"fmt"
	"io"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// Lister retrieves tmux sessions.
type Lister func() ([]TmuxSession, error)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show team and session status",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runStatus(os.Stdout, ListSessions)
	},
}

func runStatus(w io.Writer, list Lister) error {
	sessions, err := list()
	if err != nil {
		return fmt.Errorf("listing tmux sessions: %w", err)
	}

	active := make(map[string][]SessionInfo)
	for _, s := range sessions {
		team, repo, ok := ParseSessionName(s.Name, cfg.Teams)
		if !ok {
			continue
		}
		active[team] = append(active[team], SessionInfo{
			TmuxSession: s,
			Team:        team,
			Repo:        repo,
		})
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "TEAM\tPROFILE\tSESSION\tDIRECTORY\tSTATUS")

	for _, team := range sortedKeys(cfg.Teams) {
		profile := cfg.Teams[team].Profile
		infos := active[team]
		if len(infos) == 0 {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t-\t-\t-\n", team, profile)
			continue
		}
		sort.Slice(infos, func(i, j int) bool {
			return infos[i].Name < infos[j].Name
		})
		for _, info := range infos {
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
				team,
				profile,
				info.Name,
				info.Path,
				sessionStatus(info),
			)
		}
	}

	return tw.Flush()
}

func sessionStatus(info SessionInfo) string {
	if info.Attached {
		return "attached"
	}
	idle := time.Since(info.Activity)
	if idle < time.Minute {
		return "active"
	}
	return fmt.Sprintf("idle %s", formatDuration(idle))
}

func formatDuration(d time.Duration) string {
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func sortedKeys[K ~string, V any](m map[K]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)
	return keys
}
