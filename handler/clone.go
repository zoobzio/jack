// Package handler implements jack's CLI command handlers — clone, in, out,
// kill, status, and the identity resolution they share — translating cobra
// invocations into calls against the core application boundaries.
package handler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zoobzio/jack/config"
	"github.com/zoobzio/jack/core"
	"github.com/zoobzio/jack/domain"
)

// Clone builds the `jack clone` command and mounts it onto the app's root
// command.
func Clone(app *core.App) {
	cmd := &cobra.Command{
		Use:   "clone <url>",
		Short: "Clone a repo for an agent",
		Long:  "Clone a git repo into each agent's isolated workspace and apply agent config.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			names, _ := cmd.Flags().GetStringSlice("agent")
			force, _ := cmd.Flags().GetBool("force")

			agents := make([]domain.Agent, len(names))
			for i, n := range names {
				agents[i] = domain.Agent(n)
			}
			return clone(cmd.Context(), app, args[0], agents, force)
		},
	}
	cmd.Flags().StringSliceP("agent", "a", nil, "agents to clone for (required, repeatable)")
	_ = cmd.MarkFlagRequired("agent")
	cmd.Flags().BoolP("force", "f", false, "remove existing repo and session before cloning")
	app.Root().AddCommand(cmd)
}

// clone builds the base image once, then clones url into each agent's workspace,
// configures that agent's git identity, applies its config, and records it in
// the registry. An agent whose clone already exists is skipped unless force is
// set, in which case its session is killed and the clone is replaced.
func clone(ctx context.Context, app *core.App, url string, agents []domain.Agent, force bool) error {
	repo, err := domain.NewRepo(url)
	if err != nil {
		return err
	}

	if buildErr := app.Docker().Build(ctx); buildErr != nil {
		return fmt.Errorf("building jack image: %w", buildErr)
	}

	reg, err := config.NewRegistry(app.Env().RegistryPath)
	if err != nil {
		return fmt.Errorf("loading registry: %w", err)
	}

	for _, agent := range agents {
		profile, ok := app.Config().Profiles[agent]
		if !ok {
			return fmt.Errorf("unknown agent %q (no matching profile)", agent)
		}

		id, err := domain.NewIdentity(agent, repo)
		if err != nil {
			return err
		}

		dir := filepath.Join(app.Env().DataDir, string(agent), string(repo))

		// Replace an existing clone only with --force; otherwise skip it.
		if _, err := os.Stat(dir); err == nil {
			if !force {
				fmt.Printf("warning: %s already exists for agent %s, skipping (use --force to replace)\n", repo, agent)
				continue
			}
			if has, _ := app.Tmux().Has(ctx, id.Session); has {
				if kerr := app.Tmux().Kill(ctx, id.Session); kerr != nil {
					return fmt.Errorf("killing session %s: %w", id.Session, kerr)
				}
			}
			if rerr := os.RemoveAll(dir); rerr != nil {
				return fmt.Errorf("removing %s: %w", dir, rerr)
			}
		}

		if err := os.MkdirAll(filepath.Dir(dir), 0o750); err != nil {
			return fmt.Errorf("creating directory %s: %w", filepath.Dir(dir), err)
		}

		if err := app.Git().Clone(ctx, url, dir); err != nil {
			return fmt.Errorf("cloning %s for agent %s: %w", repo, agent, err)
		}

		// Configure the agent's git identity in its clone.
		if profile.Git.Name != "" {
			_ = app.Git().Config(ctx, dir, "user.name", profile.Git.Name)
		}
		if profile.Git.Email != "" {
			_ = app.Git().Config(ctx, dir, "user.email", profile.Git.Email)
		}

		if err := app.Env().ApplyAgent(agent); err != nil {
			return fmt.Errorf("applying agent %s: %w", agent, err)
		}

		reg.Add(agent, repo, url)
		if err := reg.Save(); err != nil {
			return fmt.Errorf("saving registry: %w", err)
		}

		fmt.Printf("cloned %s for agent %s\n", repo, agent)
	}

	return nil
}
