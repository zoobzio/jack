package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/zoobzio/jack/domain"
)

// Config represents the top-level YAML configuration.
type Config struct {
	Profiles   map[domain.Agent]Profile `yaml:"profiles"`
	CA         CAConfig                 `yaml:"ca"`
	Model      string                   `yaml:"model"`      // default Claude model when a profile sets none
	Permission Permission               `yaml:"permission"` // default permission mode when a profile sets none
}

// CAConfig holds certificate authority settings for agent identity.
// Containers use these values to bootstrap and issue their own certificates
// via the step CLI — jack does not manage certs on the host.
type CAConfig struct {
	URL         string `yaml:"url"`
	Fingerprint string `yaml:"fingerprint"`
	Provisioner string `yaml:"provisioner"`
}

// Profile represents a git/GitHub identity.
type Profile struct {
	Git        GitConfig    `yaml:"git"`
	GitHub     GitHubConfig `yaml:"github"`
	Model      string       `yaml:"model"`      // Claude model for this agent; overrides Config.Model, empty = Claude Code default
	Permission Permission   `yaml:"permission"` // permission mode for this agent; overrides Config.Permission
	Repos      []string     `yaml:"repos"`
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

// NewConfig reads, parses, and validates the YAML config at path. It requires
// at least one profile, and every profile name must be a valid agent name
// (see Agent.Validate) since a profile name is used as an agent identifier.
func NewConfig(path string) (*Config, error) {
	var cfg Config
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if len(cfg.Profiles) == 0 {
		return nil, errors.New("at least one profile must be defined")
	}
	if err := cfg.Permission.Validate(); err != nil {
		return nil, err
	}
	for name, profile := range cfg.Profiles {
		if err := name.Validate(); err != nil {
			return nil, err
		}
		if err := profile.Permission.Validate(); err != nil {
			return nil, fmt.Errorf("profile %q: %w", name, err)
		}
	}
	return &cfg, nil
}
