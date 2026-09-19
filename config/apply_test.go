package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyAgentCopiesRecursively(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(filepath.Join(agentSrc, "commands"), 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentSrc, "CLAUDE.md"), []byte("top-level"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentSrc, "commands", "do.md"), []byte("nested"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	env := &Env{ConfigDir: configDir, DataDir: dataDir}

	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("ApplyAgent returned error: %v", err)
	}

	dstBase := filepath.Join(dataDir, "alex", ".claude")

	if got, err := os.ReadFile(filepath.Join(dstBase, "CLAUDE.md")); err != nil { //nolint:gosec // path is under t.TempDir()
		t.Errorf("reading copied CLAUDE.md: %v", err)
	} else if string(got) != "top-level" {
		t.Errorf("CLAUDE.md content = %q, want %q", got, "top-level")
	}

	if got, err := os.ReadFile(filepath.Join(dstBase, "commands", "do.md")); err != nil { //nolint:gosec // path is under t.TempDir()
		t.Errorf("reading nested copied file: %v", err)
	} else if string(got) != "nested" {
		t.Errorf("nested file content = %q, want %q", got, "nested")
	}
}

func TestApplyAgentReplacesStaleFiles(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(agentSrc, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentSrc, "CLAUDE.md"), []byte("fresh"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Pre-create a stale file in the destination.
	dstBase := filepath.Join(dataDir, "alex", ".claude")
	if err := os.MkdirAll(dstBase, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	stale := filepath.Join(dstBase, "stale.md")
	if err := os.WriteFile(stale, []byte("stale"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	env := &Env{ConfigDir: configDir, DataDir: dataDir}

	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("ApplyAgent returned error: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale file still present (err=%v), want removed", err)
	}
	if got, err := os.ReadFile(filepath.Join(dstBase, "CLAUDE.md")); err != nil { //nolint:gosec // path is under t.TempDir()
		t.Errorf("reading copied CLAUDE.md: %v", err)
	} else if string(got) != "fresh" {
		t.Errorf("CLAUDE.md content = %q, want %q", got, "fresh")
	}
}

func TestApplyAgentKeepsDirInode(t *testing.T) {
	// A running container bind-mounts the workspace .claude directory by inode;
	// re-applying must rebuild its contents without replacing the directory, or
	// the mount is stranded on the stale copy.
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(agentSrc, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentSrc, "CLAUDE.md"), []byte("v1"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	env := &Env{ConfigDir: configDir, DataDir: dataDir}
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("first ApplyAgent: %v", err)
	}

	dstBase := filepath.Join(dataDir, "alex", ".claude")
	before, err := os.Stat(dstBase)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(agentSrc, "CLAUDE.md"), []byte("v2"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("second ApplyAgent: %v", err)
	}

	after, err := os.Stat(dstBase)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Error(".claude directory was replaced (new inode), want contents rebuilt in place")
	}
	if got, err := os.ReadFile(filepath.Join(dstBase, "CLAUDE.md")); err != nil { //nolint:gosec // path is under t.TempDir()
		t.Errorf("reading re-applied CLAUDE.md: %v", err)
	} else if string(got) != "v2" {
		t.Errorf("CLAUDE.md content = %q, want %q", got, "v2")
	}
}

func TestApplyAgentMissingSourceDir(t *testing.T) {
	env := &Env{ConfigDir: t.TempDir(), DataDir: t.TempDir()}

	err := env.ApplyAgent("ghost")
	if err == nil {
		t.Fatal("expected error for missing agent directory, got nil")
	}
}

// writeSkill lays down a skill dir with a SKILL.md carrying the given marker so
// tests can tell which source a landed skill came from.
func writeSkill(t *testing.T, base, name, marker string) {
	t.Helper()
	dir := filepath.Join(base, "skills", name)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatalf("setup skill %s: %v", name, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(marker), 0o600); err != nil {
		t.Fatalf("setup skill %s: %v", name, err)
	}
}

func TestApplyAgentFansOutSharedSkills(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(agentSrc, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// A shared skill in the top-level skills/ dir must reach the agent's workspace.
	writeSkill(t, configDir, "review", "shared-review")

	env := &Env{ConfigDir: configDir, DataDir: dataDir}
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("ApplyAgent: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dataDir, "alex", ".claude", "skills", "review", "SKILL.md")) //nolint:gosec // path under t.TempDir()
	if err != nil {
		t.Fatalf("reading fanned-out skill: %v", err)
	}
	if string(got) != "shared-review" {
		t.Errorf("skill content = %q, want %q", got, "shared-review")
	}
}

func TestApplyAgentSkillAgentOverridesShared(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(agentSrc, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Same skill name defined both by the agent and shared: the agent's wins,
	// as an atomic directory — no per-file merge.
	writeSkill(t, agentSrc, "review", "agent-review")
	writeSkill(t, configDir, "review", "shared-review")
	// A second file present only in the shared copy must NOT bleed into the
	// agent's skill: collision is resolved at directory granularity.
	if err := os.WriteFile(filepath.Join(configDir, "skills", "review", "extra.md"), []byte("shared-extra"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	env := &Env{ConfigDir: configDir, DataDir: dataDir}
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("ApplyAgent: %v", err)
	}

	dstSkill := filepath.Join(dataDir, "alex", ".claude", "skills", "review")
	got, err := os.ReadFile(filepath.Join(dstSkill, "SKILL.md")) //nolint:gosec // path under t.TempDir()
	if err != nil {
		t.Fatalf("reading skill: %v", err)
	}
	if string(got) != "agent-review" {
		t.Errorf("skill content = %q, want agent copy to win (%q)", got, "agent-review")
	}
	if _, err := os.Stat(filepath.Join(dstSkill, "extra.md")); !os.IsNotExist(err) {
		t.Errorf("shared file bled into agent skill (err=%v); collision must be per-directory", err)
	}
}

func TestApplyAgentAbsentSharedSkillsIsNoop(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(agentSrc, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentSrc, "CLAUDE.md"), []byte("soul"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}

	env := &Env{ConfigDir: configDir, DataDir: dataDir}
	// No skills/ dir exists at all: apply must still succeed and create no skills dir.
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("ApplyAgent with no shared skills: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "alex", ".claude", "skills")); !os.IsNotExist(err) {
		t.Errorf("skills dir created (err=%v), want no-op when shared skills absent", err)
	}
}

func TestApplyAgentReDropsEditedSharedSkill(t *testing.T) {
	// The refresh property: because refresh calls ApplyAgent and ApplyAgent
	// rebuilds contents in place, editing a shared skill and re-applying
	// re-drops it — the update a running container sees with no rebuild.
	configDir := t.TempDir()
	dataDir := t.TempDir()

	agentSrc := filepath.Join(configDir, "agents", "alex")
	if err := os.MkdirAll(agentSrc, 0o750); err != nil {
		t.Fatalf("setup: %v", err)
	}
	writeSkill(t, configDir, "review", "v1")

	env := &Env{ConfigDir: configDir, DataDir: dataDir}
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("first ApplyAgent: %v", err)
	}

	// Edit the shared skill, then refresh (re-apply).
	writeSkill(t, configDir, "review", "v2")
	if err := env.ApplyAgent("alex"); err != nil {
		t.Fatalf("second ApplyAgent: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dataDir, "alex", ".claude", "skills", "review", "SKILL.md")) //nolint:gosec // path under t.TempDir()
	if err != nil {
		t.Fatalf("reading re-dropped skill: %v", err)
	}
	if string(got) != "v2" {
		t.Errorf("skill content = %q, want re-dropped %q", got, "v2")
	}
}
