package jack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the top-level YAML configuration.
type Config struct {
	Profiles map[string]Profile `yaml:"profiles"`
	CA       CAConfig           `yaml:"ca"`
}

// CAConfig holds certificate authority settings for agent identity.
type CAConfig struct {
	URL         string `yaml:"url"`
	Root        string `yaml:"root"`
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

// Validate checks the Config for internal consistency.
func (c Config) Validate() error {
	if len(c.Profiles) == 0 {
		return fmt.Errorf("at least one profile must be defined")
	}
	for name := range c.Profiles {
		if strings.Contains(name, "-") {
			return fmt.Errorf("profile name %q must not contain hyphens", name)
		}
	}
	return nil
}

var cfg Config

// initConfig loads the configuration from the given YAML file.
func initConfig(configPath string) error {
	data, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}
	return nil
}
