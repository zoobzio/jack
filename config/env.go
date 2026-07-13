package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Env holds the resolved, absolute filesystem locations jack uses:
// ConfigDir for user-authored config and ConfigPath/DataDir/RegistryPath for
// jack-managed data. Paths are absolute at construction, so callers can join
// onto them directly.
type Env struct {
	ConfigDir    string // user config directory (default ~/.config/jack)
	ConfigPath   string // path to config.yaml within ConfigDir
	DataDir      string // jack-managed data directory (default ~/.jack)
	RegistryPath string // path to registry.yaml within DataDir
}

// NewEnv resolves jack's paths. Defaults are derived from the user's home
// directory and overridden by JACK_CONFIG_DIR and JACK_DATA_DIR when set.
// Override values are used verbatim, so they must be absolute paths. It errors
// only if the home directory cannot be determined.
func NewEnv() (*Env, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	configDir := filepath.Join(home, ".config", "jack")
	if v := os.Getenv("JACK_CONFIG_DIR"); v != "" {
		configDir = v
	}
	dataDir := filepath.Join(home, ".jack")
	if v := os.Getenv("JACK_DATA_DIR"); v != "" {
		dataDir = v
	}
	return &Env{
		ConfigDir:    configDir,
		ConfigPath:   filepath.Join(configDir, "config.yaml"),
		DataDir:      dataDir,
		RegistryPath: filepath.Join(dataDir, "registry.yaml"),
	}, nil
}
