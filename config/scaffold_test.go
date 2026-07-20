package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/zoobzio/jack/domain"
)

// scaffoldEnv builds an Env rooted under a fresh temp dir, matching the layout
// NewEnv would produce, so Scaffold has real paths to create.
func scaffoldEnv(t *testing.T) *Env {
	t.Helper()
	root := t.TempDir()
	configDir := filepath.Join(root, "config")
	dataDir := filepath.Join(root, "data")
	return &Env{
		ConfigDir:    configDir,
		ConfigPath:   filepath.Join(configDir, "config.yaml"),
		DataDir:      dataDir,
		RegistryPath: filepath.Join(dataDir, "registry.yaml"),
	}
}

func TestRenderSeedsIdentityAndParses(t *testing.T) {
	sc := StarterConfig{Agent: "alex", GitName: "Alex T", GitEmail: "alex@zoobz.io", GitHubUser: "zoobzio"}
	out := sc.Render()

	// The rendered file must be a valid config with the seeded profile.
	cfg, err := parseRendered(out)
	if err != nil {
		t.Fatalf("rendered config does not parse: %v", err)
	}
	profile, ok := cfg.Profiles[domain.Agent("alex")]
	if !ok {
		t.Fatalf("rendered config missing profile alex; got %v", cfg.Profiles)
	}
	if profile.Git.Name != "Alex T" || profile.Git.Email != "alex@zoobz.io" {
		t.Errorf("git identity = %+v, want Alex T / alex@zoobz.io", profile.Git)
	}
	if profile.GitHub.User != "zoobzio" {
		t.Errorf("github user = %q, want zoobzio", profile.GitHub.User)
	}
}

func TestRenderUsesPlaceholdersWhenEmpty(t *testing.T) {
	out := string(StarterConfig{Agent: "agent"}.Render())
	for _, want := range []string{"Your Name", "you@example.com", "your-github-user"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered config missing placeholder %q", want)
		}
	}
}

func TestScaffoldCreatesTree(t *testing.T) {
	env := scaffoldEnv(t)
	res, err := env.Scaffold(StarterConfig{Agent: "alex", GitName: "Alex", GitEmail: "alex@zoobz.io"})
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if !res.WroteConfig || !res.WroteSoul {
		t.Fatalf("expected fresh scaffold to write config and soul; got %+v", res)
	}

	// The directories and files jack later depends on must exist.
	mustExist(t, env.DataDir)
	mustExist(t, filepath.Join(env.ConfigDir, "projects"))
	mustExist(t, filepath.Join(env.ConfigDir, "agents", "alex"))
	mustExist(t, env.ConfigPath)
	mustExist(t, filepath.Join(env.ConfigDir, "agents", "alex", "CLAUDE.md"))

	// The written config must round-trip through the real loader.
	if _, err := NewConfig(env.ConfigPath); err != nil {
		t.Errorf("scaffolded config.yaml fails to load: %v", err)
	}
}

func TestScaffoldPreservesExistingFiles(t *testing.T) {
	env := scaffoldEnv(t)
	if _, err := env.Scaffold(StarterConfig{Agent: "alex"}); err != nil {
		t.Fatalf("first Scaffold: %v", err)
	}

	// Hand-edit the config, then re-run: the edit must survive.
	const edited = "profiles:\n  edited: {}\n"
	if err := os.WriteFile(env.ConfigPath, []byte(edited), 0o600); err != nil {
		t.Fatalf("editing config: %v", err)
	}

	res, err := env.Scaffold(StarterConfig{Agent: "alex"})
	if err != nil {
		t.Fatalf("second Scaffold: %v", err)
	}
	if res.WroteConfig || res.WroteSoul {
		t.Errorf("re-run should not rewrite existing files; got %+v", res)
	}

	got, err := os.ReadFile(env.ConfigPath) //nolint:gosec // path from test-owned Env
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(got) != edited {
		t.Errorf("config was clobbered:\n%s", got)
	}
}

// parseRendered unmarshals rendered config bytes without the profile-count
// validation NewConfig imposes, so Render can be checked in isolation.
func parseRendered(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}
