//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRootCmdMetadata(t *testing.T) {
	jtesting.AssertEqual(t, rootCmd.Use, "jack")
	jtesting.AssertEqual(t, strings.Contains(rootCmd.Short, "multi-agent"), true)
}

func TestPersistentPreRunELoadsConfig(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	_ = os.WriteFile(configPath, []byte("profiles:\n  blue:\n    git:\n      name: Rock\n      email: rock@example.com\n"), 0o600)

	t.Setenv("JACK_CONFIG_DIR", configDir)
	t.Setenv("JACK_DATA_DIR", t.TempDir())

	// Reset globals.
	env = Env{}
	cfg = Config{}

	err := rootCmd.PersistentPreRunE(rootCmd, nil)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, env.ConfigDir, configDir)
	jtesting.AssertEqual(t, cfg.Profiles["blue"].Git.Name, "Rock")
}

func TestPersistentPreRunEFailsMissingConfig(t *testing.T) {
	t.Setenv("JACK_CONFIG_DIR", t.TempDir())
	t.Setenv("JACK_DATA_DIR", t.TempDir())

	env = Env{}
	cfg = Config{}

	err := rootCmd.PersistentPreRunE(rootCmd, nil)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "reading config"), true)
}
