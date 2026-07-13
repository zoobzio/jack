package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zoobzio/jack/config"
	"github.com/zoobzio/jack/domain"
)

// fakeDocker/fakeTmux/fakeGit are minimal boundary implementations used to
// assert that the App accessors return exactly what was injected. Their methods
// are never called by these tests.
type fakeDocker struct{ Docker }

type fakeTmux struct{ Tmux }

type fakeGit struct{ Git }

var (
	_ Docker = fakeDocker{}
	_ Tmux   = fakeTmux{}
	_ Git    = fakeGit{}
)

func TestNewAppWithAccessors(t *testing.T) {
	env := &config.Env{}
	cfg := &config.Config{}
	d := fakeDocker{}
	tm := fakeTmux{}
	g := fakeGit{}

	app := NewAppWith(env, cfg, d, tm, g)

	if app.Env() != env {
		t.Error("Env() did not return the injected env")
	}
	if app.Config() != cfg {
		t.Error("Config() did not return the injected config")
	}
	if app.Docker() != d {
		t.Error("Docker() did not return the injected docker")
	}
	if app.Tmux() != tm {
		t.Error("Tmux() did not return the injected tmux")
	}
	if app.Git() != g {
		t.Error("Git() did not return the injected git")
	}
	if app.Root() == nil {
		t.Error("Root() returned nil")
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	yaml := `profiles:
  scout:
    git:
      name: Ada
      email: ada@example.com
ca:
  url: https://ca.example.com
`
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatalf("writing config: %v", err)
	}

	env := &config.Env{ConfigPath: cfgPath}
	app := NewAppWith(env, nil, NewDocker(), NewTmux(), NewGit())

	if app.Config() != nil {
		t.Fatal("Config() should be nil before LoadConfig")
	}
	if err := app.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	cfg := app.Config()
	if cfg == nil {
		t.Fatal("Config() is nil after LoadConfig")
	}
	if _, ok := cfg.Profiles[domain.Agent("scout")]; !ok {
		t.Errorf("expected profile 'scout' in loaded config, got %+v", cfg.Profiles)
	}
	if cfg.CA.URL != "https://ca.example.com" {
		t.Errorf("CA.URL = %q, want https://ca.example.com", cfg.CA.URL)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	env := &config.Env{ConfigPath: filepath.Join(t.TempDir(), "nope.yaml")}
	app := NewAppWith(env, nil, NewDocker(), NewTmux(), NewGit())
	if err := app.LoadConfig(); err == nil {
		t.Fatal("expected error loading a missing config file, got nil")
	}
}

func TestNewApp(t *testing.T) {
	// NewEnv reads HOME (or the JACK_* overrides); point them at temp dirs.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("JACK_CONFIG_DIR", t.TempDir())
	t.Setenv("JACK_DATA_DIR", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp returned nil app")
	}
	if app.Root() == nil {
		t.Error("Root() is nil")
	}
	if app.Env() == nil {
		t.Error("Env() is nil")
	}
	// Config is deliberately not loaded by NewApp.
	if app.Config() != nil {
		t.Error("Config() should be nil for a freshly built app")
	}
	if app.Docker() == nil || app.Tmux() == nil || app.Git() == nil {
		t.Error("real boundaries should be non-nil")
	}
}
