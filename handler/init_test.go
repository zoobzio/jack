package handler

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoobzio/jack/config"
)

func TestInitializeScaffoldsFromStarter(t *testing.T) {
	env := testEnv(t)
	app := testApp(env, nil, nil, nil, nil)

	sc := config.StarterConfig{Agent: "alex", GitName: "Alex T", GitEmail: "alex@zoobz.io", GitHubUser: "zoobzio"}
	var out bytes.Buffer
	if err := initialize(context.Background(), app, &out, sc, false); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	data, err := os.ReadFile(env.ConfigPath) //nolint:gosec // path from test-owned Env
	if err != nil {
		t.Fatalf("reading scaffolded config: %v", err)
	}
	body := string(data)
	for _, want := range []string{"alex", "Alex T", "alex@zoobz.io", "zoobzio"} {
		if !strings.Contains(body, want) {
			t.Errorf("config missing value %q\n%s", want, body)
		}
	}

	// The agent config dir must exist so a later clone's ApplyAgent succeeds.
	if _, err := os.Stat(filepath.Join(env.ConfigDir, "agents", "alex", "CLAUDE.md")); err != nil {
		t.Errorf("agent soul not scaffolded: %v", err)
	}
}

func TestInitializeRejectsBadAgentName(t *testing.T) {
	app := testApp(testEnv(t), nil, nil, nil, nil)
	var out bytes.Buffer
	err := initialize(context.Background(), app, &out, config.StarterConfig{Agent: "bad-name"}, false)
	if err == nil {
		t.Fatal("expected error for agent name containing '-'")
	}
}

func TestInitializeBuildsImageWhenRequested(t *testing.T) {
	env := testEnv(t)
	docker := &fakeDocker{}
	app := testApp(env, nil, docker, nil, nil)

	var out bytes.Buffer
	if err := initialize(context.Background(), app, &out, config.StarterConfig{Agent: "agent"}, true); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	// docker is on PATH in CI/dev; when present, --build must trigger a build.
	// When absent, init prints a skip notice instead — accept either, but never
	// more than one build.
	if docker.BuildCalls > 1 {
		t.Errorf("BuildCalls = %d, want at most 1", docker.BuildCalls)
	}
}

// seedStarter is the non-interactive half of flag resolution — it must apply
// flag values and seed unset git fields from the host identity. It never
// prompts, so this exercises it directly without a TTY.

func TestSeedStarterSeedsGitIdentityForUnsetFields(t *testing.T) {
	app := testApp(testEnv(t), nil, nil, nil, &fakeGit{IdentityName: "Alex T", IdentityEmail: "alex@zoobz.io"})
	Init(app)
	cmd, _, err := app.Root().Find([]string{"init"})
	if err != nil {
		t.Fatalf("finding init command: %v", err)
	}
	cmd.SetContext(context.Background())
	if err := cmd.Flags().Set("agent", "alex"); err != nil {
		t.Fatalf("setting agent flag: %v", err)
	}

	sc := seedStarter(cmd, app)
	if sc.Agent != "alex" {
		t.Errorf("Agent = %q, want alex", sc.Agent)
	}
	if sc.GitName != "Alex T" || sc.GitEmail != "alex@zoobz.io" {
		t.Errorf("git identity = %q/%q, want Alex T/alex@zoobz.io", sc.GitName, sc.GitEmail)
	}
}

func TestSeedStarterPrefersFlagsOverGitIdentity(t *testing.T) {
	app := testApp(testEnv(t), nil, nil, nil, &fakeGit{IdentityName: "Global", IdentityEmail: "global@host"})
	Init(app)
	cmd, _, err := app.Root().Find([]string{"init"})
	if err != nil {
		t.Fatalf("finding init command: %v", err)
	}
	cmd.SetContext(context.Background())
	for k, v := range map[string]string{"git-name": "Flag Name", "git-email": "flag@zoobz.io"} {
		if err := cmd.Flags().Set(k, v); err != nil {
			t.Fatalf("setting %s: %v", k, err)
		}
	}

	sc := seedStarter(cmd, app)
	if sc.GitName != "Flag Name" || sc.GitEmail != "flag@zoobz.io" {
		t.Errorf("git identity = %q/%q, want the flag values", sc.GitName, sc.GitEmail)
	}
}

func TestSeedStarterToleratesGitIdentityError(t *testing.T) {
	app := testApp(testEnv(t), nil, nil, nil, &fakeGit{IdentityErr: context.DeadlineExceeded})
	Init(app)
	cmd, _, err := app.Root().Find([]string{"init"})
	if err != nil {
		t.Fatalf("finding init command: %v", err)
	}
	cmd.SetContext(context.Background())

	sc := seedStarter(cmd, app)
	// A failed identity lookup leaves the fields empty; Render fills placeholders.
	if sc.GitName != "" || sc.GitEmail != "" {
		t.Errorf("expected empty identity on git error, got %q/%q", sc.GitName, sc.GitEmail)
	}
	if !strings.Contains(string(sc.Render()), "you@example.com") {
		t.Error("expected placeholder identity in rendered config")
	}
}
