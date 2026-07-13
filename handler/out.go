package handler

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zoobzio/jack/core"
	"github.com/zoobzio/jack/domain"
)

// Out builds the `jack out` command and mounts it onto the app's root command.
func Out(app *core.App) {
	cmd := &cobra.Command{
		Use:   "out [name]",
		Short: "Terminate a session",
		Long:  "Terminate a session by name or by --agent and --project flags, stopping its container.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if len(args) > 0 {
				name = args[0]
			}
			agent, _ := cmd.Flags().GetString("agent")
			project, _ := cmd.Flags().GetString("project")
			return out(cmd.Context(), app, name, domain.Agent(agent), domain.Repo(project))
		},
	}
	cmd.Flags().StringP("agent", "a", "", "agent name")
	cmd.Flags().StringP("project", "p", "", "project name")
	app.Root().AddCommand(cmd)
}

// out kills a session and stops its container.
func out(ctx context.Context, app *core.App, name string, agent domain.Agent, repo domain.Repo) error {
	// Resolve the session name from the typed flags when not given positionally.
	if name == "" && agent != "" && repo != "" {
		id, err := domain.NewIdentity(agent, repo)
		if err != nil {
			return err
		}
		name = id.Session
	}
	if name == "" {
		return fmt.Errorf("specify a session name or both --agent and --project")
	}

	has, err := app.Tmux().Has(ctx, name)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("session %q not found", name)
	}
	if err := app.Tmux().Kill(ctx, name); err != nil {
		return err
	}

	// Stop the container. When only a positional name was given, recover the
	// agent and repo from it: a session name is "agent-repo" and agents cannot
	// contain '-', so the first hyphen is the delimiter.
	if agent == "" || repo == "" {
		if i := strings.Index(name, "-"); i != -1 {
			agent, repo = domain.Agent(name[:i]), domain.Repo(name[i+1:])
		}
	}
	if agent != "" && repo != "" {
		if id, ierr := domain.NewIdentity(agent, repo); ierr == nil {
			if serr := app.Docker().Stop(ctx, id.Container); serr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not stop container %s: %v\n", id.Container, serr)
			}
		}
	}

	fmt.Printf("killed session %s\n", name)
	return nil
}
