package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoobzio/jack/domain"
)

func writeConfigFile(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("writing config file: %v", err)
	}
	return path
}

func TestNewConfigValid(t *testing.T) {
	const contents = `profiles:
  alex:
    git:
      name: Alex Example
      email: alex@example.com
    github:
      user: alexgh
    repos:
      - foo
      - bar
ca:
  url: https://ca.example.com
  fingerprint: abc123
  provisioner: jack
`
	path := writeConfigFile(t, contents)

	cfg, err := NewConfig(path)
	if err != nil {
		t.Fatalf("NewConfig returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("NewConfig returned nil config")
	}
	if len(cfg.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(cfg.Profiles))
	}
	p, ok := cfg.Profiles[domain.Agent("alex")]
	if !ok {
		t.Fatal("expected profile named 'alex'")
	}
	if p.Git.Name != "Alex Example" {
		t.Errorf("git name = %q, want %q", p.Git.Name, "Alex Example")
	}
	if p.Git.Email != "alex@example.com" {
		t.Errorf("git email = %q, want %q", p.Git.Email, "alex@example.com")
	}
	if p.GitHub.User != "alexgh" {
		t.Errorf("github user = %q, want %q", p.GitHub.User, "alexgh")
	}
	if len(p.Repos) != 2 || p.Repos[0] != "foo" || p.Repos[1] != "bar" {
		t.Errorf("repos = %v, want [foo bar]", p.Repos)
	}
	if cfg.CA.URL != "https://ca.example.com" {
		t.Errorf("ca url = %q, want %q", cfg.CA.URL, "https://ca.example.com")
	}
	if cfg.CA.Fingerprint != "abc123" {
		t.Errorf("ca fingerprint = %q, want %q", cfg.CA.Fingerprint, "abc123")
	}
	if cfg.CA.Provisioner != "jack" {
		t.Errorf("ca provisioner = %q, want %q", cfg.CA.Provisioner, "jack")
	}
}

func TestNewConfigErrors(t *testing.T) {
	tests := []struct {
		name    string
		wantErr string // substring expected in the error
	}{
		{
			name:    "missing file",
			wantErr: "reading config",
		},
		{
			name:    "malformed yaml",
			wantErr: "parsing config",
		},
		{
			name:    "no profiles key",
			wantErr: "at least one profile",
		},
		{
			name:    "empty profiles map",
			wantErr: "at least one profile",
		},
		{
			name:    "invalid agent name with dash",
			wantErr: "'-'",
		},
	}

	contentsFor := map[string]string{
		"malformed yaml": "profiles: : : not valid yaml\n  - broken",
		"no profiles key": `ca:
  url: https://ca.example.com
`,
		"empty profiles map": "profiles: {}\n",
		"invalid agent name with dash": `profiles:
  alex-bot:
    git:
      name: Alex
      email: alex@example.com
`,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.name == "missing file" {
				path = filepath.Join(t.TempDir(), "does-not-exist.yaml")
			} else {
				path = writeConfigFile(t, contentsFor[tt.name])
			}

			cfg, err := NewConfig(path)
			if err == nil {
				t.Fatalf("expected error, got nil (cfg=%+v)", cfg)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
