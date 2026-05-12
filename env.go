package jack

import (
	"os"
	"path/filepath"
)

// Env holds path overrides loaded from environment variables.
type Env struct {
	ConfigDir string
	DataDir   string
}

// loadEnv reads environment variables and returns an Env with defaults.
func loadEnv() Env {
	e := Env{
		ConfigDir: "~/.config/jack",
		DataDir:   "~/.jack",
	}
	if v := os.Getenv("JACK_CONFIG_DIR"); v != "" {
		e.ConfigDir = v
	}
	if v := os.Getenv("JACK_DATA_DIR"); v != "" {
		e.DataDir = v
	}
	return e
}

// configDir returns the expanded config directory path.
func (e *Env) configDir() string {
	return expandHome(e.ConfigDir)
}

// configPath returns the full path to the config file.
func (e *Env) configPath() string {
	return filepath.Join(e.configDir(), "config.yaml")
}

// dataDir returns the expanded data directory path.
func (e *Env) dataDir() string {
	return expandHome(e.DataDir)
}

// expandHome replaces a leading ~ with the user's home directory.
func expandHome(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}
