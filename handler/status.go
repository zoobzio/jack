package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/zoobzio/jack/config"
	"github.com/zoobzio/jack/core"
	"github.com/zoobzio/jack/domain"
)

// Status builds the `jack status` command bound to app and mounts it onto the
// app's root command.
func Status(app *core.App) {
	app.Root().AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Show agent and session status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return status(cmd.Context(), app, os.Stdout)
		},
	})
}

// status prints a table, per agent, of each cloned repo's session and container.
func status(ctx context.Context, app *core.App, w io.Writer) error {
	reg, err := config.NewRegistry(app.Env().RegistryPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	sessions, err := app.Tmux().List(ctx)
	if err != nil {
		return fmt.Errorf("listing sessions: %w", err)
	}
	byName := make(map[string]domain.Session, len(sessions))
	for _, s := range sessions {
		byName[s.Name] = s
	}

	agents := reg.Agents()
	if len(agents) == 0 {
		_, _ = fmt.Fprintln(w, "no projects cloned")
		return nil
	}

	for i, agent := range agents {
		if i > 0 {
			_, _ = fmt.Fprintln(w)
		}
		_, _ = fmt.Fprintln(w, agent)

		tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(tw, "PROJECT\tSESSION\tSTATUS\tCONTAINER")

		for _, entry := range reg.ForAgent(agent) {
			id, ierr := domain.NewIdentity(agent, entry.Repo)
			if ierr != nil {
				continue
			}

			// Container: running / stopped (exists) / "-" (gone).
			cstatus := "-"
			if running, rerr := app.Docker().Running(ctx, id.Container); running {
				cstatus = "running"
			} else if rerr == nil {
				cstatus = "stopped"
			}

			sessionCell, statusCell := "-", "not running"
			if s, ok := byName[id.Session]; ok {
				sessionCell, statusCell = id.Session, s.Status()
			}
			_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", entry.Repo, sessionCell, statusCell, cstatus)
		}
		_ = tw.Flush()
	}

	return nil
}
