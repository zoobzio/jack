package handler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/zoobzio/jack/domain"
)

// writeAgentConfig creates the agents/<name>/CLAUDE.md file that ApplyAgent
// copies into the agent's workspace, so the clone happy path can complete.
func writeAgentConfig(t *testing.T, configDir string, agent domain.Agent) {
	t.Helper()
	dir := filepath.Join(configDir, "agents", string(agent))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("mkdir agent config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# agent\n"), 0o600); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
}

func TestCloneUnknownAgent(t *testing.T) {
	env := testEnv(t)
	// Config has a profile for "alex" only.
	app := testApp(env, profileConfig("alex"), &fakeDocker{}, &fakeTmux{}, &fakeGit{})

	err := clone(context.Background(), app, "https://github.com/zoobzio/jack", []domain.Agent{"ghost"}, false)
	if err == nil {
		t.Fatal("clone for unknown agent = nil error, want error")
	}
}

func TestCloneHappyPath(t *testing.T) {
	env := testEnv(t)
	writeAgentConfig(t, env.ConfigDir, "alex")

	d := &fakeDocker{}
	g := &fakeGit{}
	app := testApp(env, profileConfig("alex"), d, &fakeTmux{}, g)

	url := "https://github.com/zoobzio/jack"
	if err := clone(context.Background(), app, url, []domain.Agent{"alex"}, false); err != nil {
		t.Fatalf("clone returned error: %v", err)
	}

	if d.BuildCalls != 1 {
		t.Errorf("Build called %d times, want 1", d.BuildCalls)
	}

	wantDir := filepath.Join(env.DataDir, "alex", "jack")
	if len(g.CloneCalls) != 1 {
		t.Fatalf("Clone called %d times, want 1", len(g.CloneCalls))
	}
	if g.CloneCalls[0].URL != url || g.CloneCalls[0].Dir != wantDir {
		t.Errorf("Clone = %+v, want URL %q dir %q", g.CloneCalls[0], url, wantDir)
	}

	// Git identity is configured for the agent's clone.
	var gotName, gotEmail bool
	for _, c := range g.ConfigCalls {
		if c.Dir != wantDir {
			t.Errorf("Config in dir %q, want %q", c.Dir, wantDir)
		}
		switch c.Key {
		case "user.name":
			gotName = c.Value == "Test Agent"
		case "user.email":
			gotEmail = c.Value == "agent@test.io"
		}
	}
	if !gotName || !gotEmail {
		t.Errorf("Config calls = %+v, want user.name and user.email set", g.ConfigCalls)
	}

	// The registry file was written.
	if _, err := os.Stat(env.RegistryPath); err != nil {
		t.Errorf("registry file not written at %s: %v", env.RegistryPath, err)
	}
}

func TestCloneExistingSkippedWithoutForce(t *testing.T) {
	env := testEnv(t)
	writeAgentConfig(t, env.ConfigDir, "alex")

	// The clone directory already exists.
	dir := filepath.Join(env.DataDir, "alex", "jack")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("pre-create dir: %v", err)
	}

	g := &fakeGit{}
	app := testApp(env, profileConfig("alex"), &fakeDocker{}, &fakeTmux{}, g)

	if err := clone(context.Background(), app, "https://github.com/zoobzio/jack", []domain.Agent{"alex"}, false); err != nil {
		t.Fatalf("clone returned error: %v", err)
	}
	if len(g.CloneCalls) != 0 {
		t.Errorf("Clone called %d times, want 0 (existing dir, no --force)", len(g.CloneCalls))
	}
}

func TestCloneForceReplacesExisting(t *testing.T) {
	env := testEnv(t)
	writeAgentConfig(t, env.ConfigDir, "alex")

	dir := filepath.Join(env.DataDir, "alex", "jack")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("pre-create dir: %v", err)
	}
	marker := filepath.Join(dir, "stale.txt")
	if err := os.WriteFile(marker, []byte("old"), 0o600); err != nil {
		t.Fatalf("write marker: %v", err)
	}

	g := &fakeGit{}
	// Has=true → the existing session is killed before replacing the clone.
	tm := &fakeTmux{HasResult: true}
	app := testApp(env, profileConfig("alex"), &fakeDocker{}, tm, g)

	if err := clone(context.Background(), app, "https://github.com/zoobzio/jack", []domain.Agent{"alex"}, true); err != nil {
		t.Fatalf("clone --force returned error: %v", err)
	}

	if len(tm.KillNames) != 1 || tm.KillNames[0] != "alex-jack" {
		t.Errorf("Kill = %v, want [alex-jack]", tm.KillNames)
	}
	if len(g.CloneCalls) != 1 {
		t.Errorf("Clone called %d times, want 1 (re-clone under --force)", len(g.CloneCalls))
	}
	// The old directory (and its stale marker) was removed.
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("stale marker still present, want removed: err=%v", err)
	}
}
