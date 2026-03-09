package jack

import (
	"context"
	"fmt"
	"strings"

	"github.com/zoobzio/flux"
	"github.com/zoobzio/flux/file"
)

// Config represents the top-level YAML configuration.
type Config struct {
	Profiles map[string]Profile `yaml:"profiles"`
	Roles    map[string]Role    `yaml:"roles"`
	Teams    map[string]Team    `yaml:"teams"`
	Matrix   MatrixConfig       `yaml:"matrix"`
}

// MatrixConfig holds Matrix homeserver connection settings.
type MatrixConfig struct {
	Homeserver        string `yaml:"homeserver"`
	RegistrationToken string `yaml:"registration_token"`
}

// Profile represents a git/GitHub/SSH identity.
type Profile struct {
	Git    GitConfig    `yaml:"git"`
	GitHub GitHubConfig `yaml:"github"`
	SSH    SSHConfig    `yaml:"ssh"`
}

// GitConfig holds git identity settings.
type GitConfig struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

// GitHubConfig holds GitHub account settings.
type GitHubConfig struct {
	User string `yaml:"user"`
}

// SSHConfig holds SSH key settings.
type SSHConfig struct {
	Key string `yaml:"key"`
}

// Role bundles a set of skills that can be assigned to a team at clone time.
type Role struct {
	Skills []string `yaml:"skills"`
}

// Team defines a named team with its profile, agents, and includes.
type Team struct {
	Profile string   `yaml:"profile"`
	Agents  []string `yaml:"agents"`
	Include []string `yaml:"include"`
}

// Validate checks the Config for internal consistency.
func (c Config) Validate() error {
	if len(c.Profiles) == 0 {
		return fmt.Errorf("at least one profile must be defined")
	}
	if len(c.Teams) == 0 {
		return fmt.Errorf("at least one team must be defined")
	}
	for name, team := range c.Teams {
		if strings.Contains(name, "-") {
			return fmt.Errorf("team name %q must not contain hyphens", name)
		}
		if _, ok := c.Profiles[team.Profile]; !ok {
			return fmt.Errorf("team %q references unknown profile %q", name, team.Profile)
		}
	}
	return nil
}

var capacitor *flux.Capacitor[Config]

var cfg Config

// initConfig creates and starts the flux capacitor for the config file.
func initConfig(ctx context.Context, configPath string) error {
	capacitor = flux.New[Config](
		file.New(configPath),
		func(_ context.Context, _, curr Config) error {
			cfg = curr
			return nil
		},
	).Codec(flux.YAMLCodec{})

	if err := capacitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	current, ok := capacitor.Current()
	if !ok {
		return fmt.Errorf("config loaded but no current value available")
	}
	cfg = current

	return nil
}
