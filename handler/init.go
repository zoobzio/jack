package handler

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"text/tabwriter"

	"github.com/charmbracelet/huh"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/zoobzio/jack/config"
	"github.com/zoobzio/jack/core"
	"github.com/zoobzio/jack/domain"
)

// Init builds the `jack init` command and mounts it onto the app's root command.
// It defines its own no-op PersistentPreRunE so the root's config-loading pre-run
// is skipped: init exists precisely to create the config that does not yet exist.
func Init(app *core.App) {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold jack's config and check host prerequisites",
		Long: "Create ~/.config/jack (config.yaml + agent/project dirs) and ~/.jack, " +
			"seeded from your global git identity, and report which host tools are present. " +
			"Flags supply values non-interactively; anything left unset is prompted for when " +
			"run in a terminal. Existing files are never overwritten.",
		Args:              cobra.NoArgs,
		PersistentPreRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(cmd *cobra.Command, _ []string) error {
			build, _ := cmd.Flags().GetBool("build")
			sc, err := resolveStarter(cmd, app)
			if err != nil {
				return err
			}
			return initialize(cmd.Context(), app, os.Stdout, sc, build)
		},
	}
	cmd.Flags().StringP("agent", "a", "agent", "name of the first agent profile")
	cmd.Flags().String("git-name", "", "git user.name for the first profile (default: global git identity)")
	cmd.Flags().String("git-email", "", "git user.email for the first profile (default: global git identity)")
	cmd.Flags().String("github", "", "GitHub user for the first profile")
	cmd.Flags().Bool("build", false, "also build the base Docker image now")
	app.Root().AddCommand(cmd)
}

// prerequisites are the host CLIs jack shells out to; init reports which are on
// PATH so a fresh operator knows what is still missing.
var prerequisites = []string{"docker", "tmux", "git"}

// resolveStarter builds the StarterConfig for init from flags, then fills any
// field the user did not pass. It seeds git identity from the host and, when
// attached to a terminal, prompts for the remaining gaps with the flag/seeded
// values prefilled. Without a TTY (scripts, CI) it keeps those defaults, so init
// stays fully non-interactive when every value comes from a flag.
func resolveStarter(cmd *cobra.Command, app *core.App) (config.StarterConfig, error) {
	sc := seedStarter(cmd, app)
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		return sc, nil
	}

	f := cmd.Flags()
	var fields []huh.Field
	if !f.Changed("agent") {
		fields = append(fields, huh.NewInput().
			Title("Agent name").
			Description("identifier for the first profile (no '-')").
			Value(&sc.Agent).
			Validate(func(s string) error { return domain.Agent(s).Validate() }))
	}
	if !f.Changed("git-name") {
		fields = append(fields, huh.NewInput().Title("Git name").Value(&sc.GitName))
	}
	if !f.Changed("git-email") {
		fields = append(fields, huh.NewInput().Title("Git email").Value(&sc.GitEmail))
	}
	if !f.Changed("github") {
		fields = append(fields, huh.NewInput().Title("GitHub user").Value(&sc.GitHubUser))
	}
	if len(fields) == 0 {
		return sc, nil
	}
	if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
		return config.StarterConfig{}, fmt.Errorf("collecting init details: %w", err)
	}
	return sc, nil
}

// seedStarter reads the init flags into a StarterConfig, filling unset git
// identity fields from the host's global git config. It does no prompting, so
// the same seeded values back both the interactive prompts (as prefilled
// defaults) and the non-interactive path (as final values).
func seedStarter(cmd *cobra.Command, app *core.App) config.StarterConfig {
	f := cmd.Flags()
	agent, _ := f.GetString("agent")
	gitName, _ := f.GetString("git-name")
	gitEmail, _ := f.GetString("git-email")
	github, _ := f.GetString("github")

	if !f.Changed("git-name") || !f.Changed("git-email") {
		// A missing git binary or unset identity just leaves the fields empty,
		// which Render turns into editable placeholders.
		name, email, _ := app.Git().GlobalIdentity(cmd.Context())
		if !f.Changed("git-name") {
			gitName = name
		}
		if !f.Changed("git-email") {
			gitEmail = email
		}
	}

	return config.StarterConfig{Agent: agent, GitName: gitName, GitEmail: gitEmail, GitHubUser: github}
}

// initialize runs the preflight check, scaffolds the config and data trees from
// the resolved StarterConfig, and prints next steps. With build set it also
// builds the base image. It never overwrites existing config files.
func initialize(ctx context.Context, app *core.App, w io.Writer, sc config.StarterConfig, build bool) error {
	if err := domain.Agent(sc.Agent).Validate(); err != nil {
		return err
	}

	missing := preflight(w)

	res, err := app.Env().Scaffold(sc)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(w)
	report(w, res.WroteConfig, res.ConfigPath, "left untouched (already exists)")
	report(w, res.WroteSoul, res.AgentDir, "agent config already present")

	if build {
		if slices.Contains(missing, "docker") {
			_, _ = fmt.Fprintln(w, "\nskipping --build: docker is not installed")
		} else {
			_, _ = fmt.Fprintln(w, "\nbuilding base image...")
			if berr := app.Docker().Build(ctx); berr != nil {
				return fmt.Errorf("building base image: %w", berr)
			}
		}
	}

	printNextSteps(w, domain.Agent(sc.Agent), res.WroteConfig, missing)
	return nil
}

// preflight prints a presence check for each prerequisite and returns the ones
// not found on PATH, so the caller can tailor the follow-up guidance.
func preflight(w io.Writer) []string {
	_, _ = fmt.Fprintln(w, "checking host prerequisites:")
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var missing []string
	for _, tool := range prerequisites {
		mark := "ok"
		if _, err := exec.LookPath(tool); err != nil {
			mark, missing = "MISSING", append(missing, tool)
		}
		_, _ = fmt.Fprintf(tw, "  %s\t%s\n", tool, mark)
	}
	_ = tw.Flush()
	return missing
}

// report prints a one-line result for a scaffolded path: created, or the given
// reason it was left as-is.
func report(w io.Writer, created bool, path, keptReason string) {
	if created {
		_, _ = fmt.Fprintf(w, "created %s\n", path)
		return
	}
	_, _ = fmt.Fprintf(w, "%s — %s\n", path, keptReason)
}

// printNextSteps closes init with guidance: install anything missing, edit the
// fresh config, and clone a repo for the agent.
func printNextSteps(w io.Writer, agent domain.Agent, wroteConfig bool, missing []string) {
	_, _ = fmt.Fprintln(w, "\nnext steps:")
	if len(missing) > 0 {
		_, _ = fmt.Fprintf(w, "  - install missing tools: %v\n", missing)
	}
	if wroteConfig {
		_, _ = fmt.Fprintf(w, "  - review the %s profile in your config.yaml\n", agent)
	}
	_, _ = fmt.Fprintf(w, "  - jack clone <url> --agent %s\n", agent)
}
