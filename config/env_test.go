package config

import (
	"path/filepath"
	"testing"
)

func TestNewEnvWithOverrides(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("JACK_CONFIG_DIR", configDir)
	t.Setenv("JACK_DATA_DIR", dataDir)

	env, err := NewEnv()
	if err != nil {
		t.Fatalf("NewEnv returned error: %v", err)
	}

	if env.ConfigDir != configDir {
		t.Errorf("ConfigDir = %q, want %q", env.ConfigDir, configDir)
	}
	if env.DataDir != dataDir {
		t.Errorf("DataDir = %q, want %q", env.DataDir, dataDir)
	}
	if want := filepath.Join(configDir, "config.yaml"); env.ConfigPath != want {
		t.Errorf("ConfigPath = %q, want %q", env.ConfigPath, want)
	}
	if want := filepath.Join(dataDir, "registry.yaml"); env.RegistryPath != want {
		t.Errorf("RegistryPath = %q, want %q", env.RegistryPath, want)
	}
}

func TestNewEnvDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Ensure no override env leaks in from the host.
	t.Setenv("JACK_CONFIG_DIR", "")
	t.Setenv("JACK_DATA_DIR", "")

	env, err := NewEnv()
	if err != nil {
		t.Fatalf("NewEnv returned error: %v", err)
	}

	wantConfigDir := filepath.Join(home, ".config", "jack")
	wantDataDir := filepath.Join(home, ".jack")

	if env.ConfigDir != wantConfigDir {
		t.Errorf("ConfigDir = %q, want %q", env.ConfigDir, wantConfigDir)
	}
	if env.DataDir != wantDataDir {
		t.Errorf("DataDir = %q, want %q", env.DataDir, wantDataDir)
	}
	if want := filepath.Join(wantConfigDir, "config.yaml"); env.ConfigPath != want {
		t.Errorf("ConfigPath = %q, want %q", env.ConfigPath, want)
	}
	if want := filepath.Join(wantDataDir, "registry.yaml"); env.RegistryPath != want {
		t.Errorf("RegistryPath = %q, want %q", env.RegistryPath, want)
	}
}
