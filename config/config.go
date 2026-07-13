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
	Profiles map[domain.Agent]Profile `yaml:"profiles"`
	CA       CAConfig                 `yaml:"ca"`
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
	Git    GitConfig    `yaml:"git"`
	GitHub GitHubConfig `yaml:"github"`
	Repos  []string     `yaml:"repos"`
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
	for name := range cfg.Profiles {
		err := name.Validate()
		if err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}
