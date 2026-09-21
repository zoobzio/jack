package handler

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zoobzio/jack/config"
	"github.com/zoobzio/jack/domain"
)

// registerClone records repo as cloned for agent in the env's registry and
// creates the clone directory on disk — the two facts in requires before it
// will enter an agent-repo.
func registerClone(t *testing.T, env *config.Env, agent domain.Agent, repo domain.Repo) {
	t.Helper()
	reg, err := config.NewRegistry(env.RegistryPath)
	if err != nil {
		t.Fatal(err)
	}
	reg.Add(agent, repo, "https://host/u/"+string(repo)+".git")
	if err := reg.Save(); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(env.DataDir, string(agent), string(repo)), 0o750); err != nil {
		t.Fatal(err)
	}
}

func TestInRefusesUnregisteredProject(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)

	// "jack" was cloned for alex, but "other" was not: in must refuse rather
	// than build a container around a workspace that does not exist.
	registerClone(t, env, "alex", "jack")
	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningErr: errors.New("no such container")}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	err := in(context.Background(), app, "alex", "other", false, false)
	if err == nil {
		t.Fatal("in succeeded for a project that was never cloned")
	}
	if !strings.Contains(err.Error(), "not cloned") {
		t.Errorf("error = %q, want it to say the project is not cloned", err)
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times for an uncloned project, want 0", len(d.RunSpecs))
	}
	if len(tm.CreateCalls) != 0 || len(tm.AttachNames) != 0 {
		t.Errorf("tmux touched for an uncloned project: create=%v attach=%v", tm.CreateCalls, tm.AttachNames)
	}
}

func TestInRefusesUnregisteredAgent(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)

	// bob has a profile but nothing cloned; an explicit --agent must not bypass
	// the registry.
	registerClone(t, env, "alex", "jack")
	cfg := profileConfig("alex")
	cfg.Profiles["bob"] = cfg.Profiles["alex"]
	d := &fakeDocker{RunningErr: errors.New("no such container")}
	app := testApp(env, cfg, d, &fakeTmux{HasResult: false}, &fakeGit{})

	if err := in(context.Background(), app, "bob", "jack", false, false); err == nil {
		t.Fatal("in succeeded for an agent with no clone of the project")
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times for an uncloned agent-repo, want 0", len(d.RunSpecs))
	}
}

func TestInRefusesMissingCloneDir(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)

	// Registered, but the clone directory is gone from disk (deleted by hand):
	// docker would silently bind-mount a fresh empty directory in its place.
	registerClone(t, env, "alex", "jack")
	if err := os.RemoveAll(filepath.Join(env.DataDir, "alex", "jack")); err != nil {
		t.Fatal(err)
	}
	d := &fakeDocker{RunningErr: errors.New("no such container")}
	app := testApp(env, profileConfig("alex"), d, &fakeTmux{HasResult: false}, &fakeGit{})

	err := in(context.Background(), app, "alex", "jack", false, false)
	if err == nil {
		t.Fatal("in succeeded with the clone directory missing")
	}
	if !strings.Contains(err.Error(), "missing") {
		t.Errorf("error = %q, want it to say the clone is missing", err)
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times with the clone missing, want 0", len(d.RunSpecs))
	}
}

func TestInSessionExistsAttaches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	tm := &fakeTmux{HasResult: true}
	d := &fakeDocker{}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	if len(tm.AttachNames) != 1 || tm.AttachNames[0] != "alex-jack" {
		t.Errorf("Attach = %v, want [alex-jack]", tm.AttachNames)
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times, want 0 for an existing session", len(d.RunSpecs))
	}
	if len(tm.CreateCalls) != 0 {
		t.Errorf("Create called for an existing session: %v", tm.CreateCalls)
	}
}

func TestInStartsContainerAndCreatesSession(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	// Session absent and container nonexistent (Running errors for a container
	// that does not exist).
	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningErr: errors.New("no such container")}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	if len(d.StopNames) != 0 {
		t.Errorf("Stop called for a nonexistent container: %v", d.StopNames)
	}
	if len(d.RunSpecs) != 1 {
		t.Fatalf("Run called %d times, want 1", len(d.RunSpecs))
	}
	if d.RunSpecs[0].Name != "jack-alex-jack" {
		t.Errorf("Run spec name = %q, want jack-alex-jack", d.RunSpecs[0].Name)
	}
	if len(tm.CreateCalls) != 1 || tm.CreateCalls[0].Name != "alex-jack" {
		t.Errorf("Create = %v, want one call for alex-jack", tm.CreateCalls)
	}
	if len(tm.AttachNames) != 1 || tm.AttachNames[0] != "alex-jack" {
		t.Errorf("Attach = %v, want [alex-jack]", tm.AttachNames)
	}
	// CA.URL is empty and no setup scripts exist, so nothing was exec'd.
	if len(d.ExecCalls) != 0 {
		t.Errorf("Exec called %d times, want 0", len(d.ExecCalls))
	}

	// A fresh start seeds the agent's private Claude state before the mounts.
	if _, err := os.Stat(env.ClaudeDir("alex")); err != nil {
		t.Errorf("agent claude dir not seeded: %v", err)
	}
	if _, err := os.Stat(env.ClaudeJSON("alex")); err != nil {
		t.Errorf("agent claude.json not seeded: %v", err)
	}
	// …and renders the session env the spec mounts, even with no secrets.
	if _, err := os.Stat(env.SessionEnv("alex")); err != nil {
		t.Errorf("session env not rendered: %v", err)
	}

	// The launch sources the session env before exec'ing claude, so a later
	// `jack refresh` reaches the next launch without recreating the container.
	cmd := tm.CreateCalls[0].Cmd
	if !strings.Contains(cmd, ". /root/.jack/session.env; exec claude") {
		t.Errorf("launch cmd = %q, want it to source the session env then exec claude", cmd)
	}

	// claude launches with --add-dir on the workspace parent so it discovers
	// skills applied at /root/workspace/.claude/skills, which sit above the repo
	// WORKDIR (the git root) where the project-skill walk would otherwise stop.
	if !strings.Contains(cmd, "--add-dir /root/workspace") {
		t.Errorf("launch cmd = %q, want --add-dir /root/workspace so skills above the repo root are discovered", cmd)
	}
}

func TestInInjectsAgentSecrets(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	// A secrets file for the agent lands in the container env on fresh start.
	if err := os.MkdirAll(filepath.Join(env.DataDir, "secrets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env.SecretsPath("alex"), []byte("GH_TOKEN=ghp_alex\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	d := &fakeDocker{RunningErr: errors.New("no such container")}
	app := testApp(env, profileConfig("alex"), d, &fakeTmux{HasResult: false}, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
		t.Fatalf("in returned error: %v", err)
	}
	if got := d.RunSpecs[0].Env["GH_TOKEN"]; got != "ghp_alex" {
		t.Errorf("GH_TOKEN = %q, want ghp_alex", got)
	}
}

func TestInReseedRelinksCredentials(t *testing.T) {
	home := claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	// Session already exists, so in only attaches — but --reseed must still
	// relink the agent's credentials from the host login first.
	tm := &fakeTmux{HasResult: true}
	app := testApp(env, profileConfig("alex"), &fakeDocker{}, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", true, false); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	ai, err := os.Stat(filepath.Join(env.ClaudeDir("alex"), ".credentials.json"))
	if err != nil {
		t.Fatalf("agent credentials missing after reseed: %v", err)
	}
	hi, err := os.Stat(filepath.Join(home, ".claude", ".credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(ai, hi) {
		t.Error("reseeded credentials are not hard-linked to the host's")
	}
	if len(tm.AttachNames) != 1 {
		t.Errorf("Attach = %v, want one call", tm.AttachNames)
	}
}

func TestInRemovesStoppedContainerBeforeRun(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	// Session absent; the container exists but is stopped (e.g. after a host
	// reboot): Running reports (false, nil). The remnant must be removed before
	// the fresh run, or `docker run --name` would collide with it.
	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningResult: false, RunningErr: nil}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	if len(d.StopNames) != 1 || d.StopNames[0] != "jack-alex-jack" {
		t.Errorf("Stop = %v, want [jack-alex-jack] to clear the stopped remnant", d.StopNames)
	}
	if len(d.RunSpecs) != 1 {
		t.Fatalf("Run called %d times, want 1", len(d.RunSpecs))
	}
	if len(tm.AttachNames) != 1 || tm.AttachNames[0] != "alex-jack" {
		t.Errorf("Attach = %v, want [alex-jack]", tm.AttachNames)
	}
}

func TestInStoppedContainerRemovalFails(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	// Removing the stopped remnant fails: in must surface the error rather than
	// attempt a doomed `docker run`.
	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningResult: false, StopErr: errors.New("permission denied")}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, false); err == nil {
		t.Fatal("in succeeded despite failing to remove the stopped container")
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times after a failed removal, want 0", len(d.RunSpecs))
	}
}

func TestInModelResolution(t *testing.T) {
	// The top-level default reaches the container when the profile sets no model.
	t.Run("default", func(t *testing.T) {
		claudeHome(t)
		cfg := &config.Config{
			Model:    "claude-sonnet-5",
			Profiles: map[domain.Agent]config.Profile{"alex": {}},
		}
		env := testEnv(t)
		registerClone(t, env, "alex", "jack")
		d := &fakeDocker{RunningErr: errors.New("no such container")}
		app := testApp(env, cfg, d, &fakeTmux{HasResult: false}, &fakeGit{})

		if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
			t.Fatalf("in returned error: %v", err)
		}
		if got := d.RunSpecs[0].Env["ANTHROPIC_MODEL"]; got != "claude-sonnet-5" {
			t.Errorf("ANTHROPIC_MODEL = %q, want claude-sonnet-5 (top-level default)", got)
		}
	})

	// A per-profile model overrides the top-level default.
	t.Run("override", func(t *testing.T) {
		claudeHome(t)
		cfg := &config.Config{
			Model:    "claude-sonnet-5",
			Profiles: map[domain.Agent]config.Profile{"alex": {Model: "claude-opus-4-8"}},
		}
		env := testEnv(t)
		registerClone(t, env, "alex", "jack")
		d := &fakeDocker{RunningErr: errors.New("no such container")}
		app := testApp(env, cfg, d, &fakeTmux{HasResult: false}, &fakeGit{})

		if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
			t.Fatalf("in returned error: %v", err)
		}
		if got := d.RunSpecs[0].Env["ANTHROPIC_MODEL"]; got != "claude-opus-4-8" {
			t.Errorf("ANTHROPIC_MODEL = %q, want claude-opus-4-8 (profile override)", got)
		}
	})
}

func TestInPermissionResolution(t *testing.T) {
	// The top-level default flows into the launch command when the profile is bare.
	t.Run("default", func(t *testing.T) {
		claudeHome(t)
		cfg := &config.Config{
			Permission: config.PermissionBypass,
			Profiles:   map[domain.Agent]config.Profile{"alex": {}},
		}
		env := testEnv(t)
		registerClone(t, env, "alex", "jack")
		tm := &fakeTmux{HasResult: false}
		app := testApp(env, cfg, &fakeDocker{RunningErr: errors.New("no such container")}, tm, &fakeGit{})

		if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
			t.Fatalf("in returned error: %v", err)
		}
		if len(tm.CreateCalls) != 1 {
			t.Fatalf("Create calls = %d, want 1", len(tm.CreateCalls))
		}
		if !strings.Contains(tm.CreateCalls[0].Cmd, "--dangerously-skip-permissions") {
			t.Errorf("launch cmd = %q, want it to contain --dangerously-skip-permissions", tm.CreateCalls[0].Cmd)
		}
	})

	// A per-profile permission overrides the top-level default.
	t.Run("override", func(t *testing.T) {
		claudeHome(t)
		cfg := &config.Config{
			Permission: config.PermissionBypass,
			Profiles:   map[domain.Agent]config.Profile{"alex": {Permission: config.PermissionAcceptEdits}},
		}
		env := testEnv(t)
		registerClone(t, env, "alex", "jack")
		tm := &fakeTmux{HasResult: false}
		app := testApp(env, cfg, &fakeDocker{RunningErr: errors.New("no such container")}, tm, &fakeGit{})

		if err := in(context.Background(), app, "alex", "jack", false, false); err != nil {
			t.Fatalf("in returned error: %v", err)
		}
		cmd := tm.CreateCalls[0].Cmd
		if !strings.Contains(cmd, "--permission-mode acceptEdits") {
			t.Errorf("launch cmd = %q, want --permission-mode acceptEdits", cmd)
		}
		if strings.Contains(cmd, "dangerously") {
			t.Errorf("launch cmd = %q, should not contain the bypass flag", cmd)
		}
	})
}

func TestInUpdateExistingSessionUpgradesThenAttaches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	tm := &fakeTmux{HasResult: true}
	d := &fakeDocker{}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, true); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	// The session's container is running, so the update execs into it before
	// the attach; no container is started.
	if len(d.ExecCalls) != 1 || d.ExecCalls[0].Name != "jack-alex-jack" {
		t.Fatalf("Exec = %v, want one update call into jack-alex-jack", d.ExecCalls)
	}
	if got := strings.Join(d.ExecCalls[0].Cmd, " "); !strings.Contains(got, "@anthropic-ai/claude-code@latest") {
		t.Errorf("Exec cmd = %q, want it to install @anthropic-ai/claude-code@latest", got)
	}
	if len(d.RunSpecs) != 0 {
		t.Errorf("Run called %d times, want 0 for an existing session", len(d.RunSpecs))
	}
	if len(tm.AttachNames) != 1 || tm.AttachNames[0] != "alex-jack" {
		t.Errorf("Attach = %v, want [alex-jack]", tm.AttachNames)
	}
}

func TestInUpdateFreshContainerUpgradesBeforeSession(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningErr: errors.New("no such container")}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	if err := in(context.Background(), app, "alex", "jack", false, true); err != nil {
		t.Fatalf("in returned error: %v", err)
	}

	if len(d.RunSpecs) != 1 {
		t.Fatalf("Run called %d times, want 1", len(d.RunSpecs))
	}
	// No CA and no setup scripts, so the update is the only exec, and it lands
	// in the container that was just started.
	if len(d.ExecCalls) != 1 || d.ExecCalls[0].Name != "jack-alex-jack" {
		t.Fatalf("Exec = %v, want one update call into jack-alex-jack", d.ExecCalls)
	}
	if got := strings.Join(d.ExecCalls[0].Cmd, " "); !strings.Contains(got, "@anthropic-ai/claude-code@latest") {
		t.Errorf("Exec cmd = %q, want it to install @anthropic-ai/claude-code@latest", got)
	}
	if len(tm.CreateCalls) != 1 {
		t.Errorf("Create = %v, want one call", tm.CreateCalls)
	}
}

func TestInUpdateFailureLeavesContainerRunning(t *testing.T) {
	claudeHome(t)
	env := testEnv(t)
	registerClone(t, env, "alex", "jack")

	tm := &fakeTmux{HasResult: false}
	d := &fakeDocker{RunningErr: errors.New("no such container"), ExecErr: errors.New("npm: network unreachable")}
	app := testApp(env, profileConfig("alex"), d, tm, &fakeGit{})

	err := in(context.Background(), app, "alex", "jack", false, true)
	if err == nil {
		t.Fatal("in succeeded despite a failed update")
	}
	if !strings.Contains(err.Error(), "updating claude code") {
		t.Errorf("error = %q, want it to name the update step", err)
	}
	// The container still has a working (old) claude, so it is kept for the
	// next `jack in`; no session is created around the failed attempt.
	if len(d.StopNames) != 0 {
		t.Errorf("Stop called after a failed update: %v", d.StopNames)
	}
	if len(tm.CreateCalls) != 0 || len(tm.AttachNames) != 0 {
		t.Errorf("tmux touched after a failed update: create=%v attach=%v", tm.CreateCalls, tm.AttachNames)
	}
}
