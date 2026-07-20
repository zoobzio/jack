// Package core wires the jack application together: it resolves the environment
// and config, holds the Docker, Tmux, and Git boundaries to the host CLIs, and
// builds the container Spec that describes each agent-repo session.
package core

import (
	"github.com/spf13/cobra"

	"github.com/zoobzio/jack/config"
)

// App is the wired jack application: it holds the resolved environment, the
// loaded config, the Docker, Tmux, and Git boundaries, and the root cobra
// command. Command handlers in the handler package take an *App and mount
// themselves onto its root via Root().AddCommand; cmd/jack builds one with the
// real boundaries, registers the handlers, and calls Execute.
type App struct {
	env    *config.Env
	config *config.Config
	docker Docker
	tmux   Tmux
	git    Git
	root   *cobra.Command
}

// NewApp builds an App with the real boundaries and resolved paths. Config is
// deliberately not loaded here: it is loaded in the root command's pre-run (see
// LoadConfig), so that `jack --help` and a config-less first run do not fail on
// a missing config file.
func NewApp() (*App, error) {
	env, err := config.NewEnv()
	if err != nil {
		return nil, err
	}
	return NewAppWith(env, nil, NewDocker(), NewTmux(), NewGit()), nil
}

// NewAppWith assembles an App from explicitly provided parts and builds its root
// command. NewApp uses it with the real boundaries and a nil config (loaded
// later); tests use it with fakes and a config built in-memory.
func NewAppWith(env *config.Env, cfg *config.Config, docker Docker, tmux Tmux, git Git) *App {
	a := &App{
		env:    env,
		config: cfg,
		docker: docker,
		tmux:   tmux,
		git:    git,
	}
	a.root = &cobra.Command{
		Use:           "jack",
		Short:         "Operator console for multi-agent development",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Config is loaded once, before any subcommand runs; cobra skips this
		// for --help, so `jack --help` works without a config file.
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return a.LoadConfig()
		},
	}
	return a
}

// Root returns the root command so handlers can mount their subcommands onto it.
func (a *App) Root() *cobra.Command { return a.root }

// Execute runs the root command.
func (a *App) Execute() error { return a.root.Execute() }

// LoadConfig reads and validates the config at the resolved path and stores it
// on the app. It is called once from the root command's PersistentPreRunE,
// which cobra skips for --help.
func (a *App) LoadConfig() error {
	cfg, err := config.NewConfig(a.env.ConfigPath)
	if err != nil {
		return err
	}
	a.config = cfg
	return nil
}

// Env returns the resolved environment paths.
func (a *App) Env() *config.Env { return a.env }

// Config returns the loaded config, or nil before LoadConfig has run.
func (a *App) Config() *config.Config { return a.config }

// Docker returns the docker boundary.
func (a *App) Docker() Docker { return a.docker }

// Tmux returns the tmux boundary.
func (a *App) Tmux() Tmux { return a.tmux }

// Git returns the git boundary.
func (a *App) Git() Git { return a.git }
