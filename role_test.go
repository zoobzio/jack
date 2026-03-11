//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestApplyTeamUnknownTeam(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/tmp/jack"}
	err := applyTeam("bogus", "vicky", "/tmp/repo", noopCopier)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "team skills directory"), true)
}

func TestApplyTeamCopiesAllCategories(t *testing.T) {
	// Set up a config directory with governance, team skills, orders, projects.
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/tmp/jack"}

	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("be good"), 0o600)

	// Team skills directory with skill dirs.
	teamSkillsDir := filepath.Join(configDir, "teams", "blue", "skills")
	_ = os.MkdirAll(filepath.Join(teamSkillsDir, "commit"), 0o750)
	_ = os.WriteFile(filepath.Join(teamSkillsDir, "commit", "SKILL.md"), []byte("commit"), 0o600)
	_ = os.MkdirAll(filepath.Join(teamSkillsDir, "pr"), 0o750)
	_ = os.WriteFile(filepath.Join(teamSkillsDir, "pr", "SKILL.md"), []byte("pr"), 0o600)

	// Team ORDERS.md.
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "ORDERS.md"), []byte("orders"), 0o600)

	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("mission"), 0o600)

	// Team agent directory.
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue", "agents"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "agents", "zidgel.md"), []byte("zidgel"), 0o600)

	type copyOp struct{ src, dst string }
	var copies []copyOp

	copier := func(src, dst string) error {
		copies = append(copies, copyOp{src, dst})
		return nil
	}

	dir := t.TempDir()
	err := applyTeam("blue", "vicky", dir, copier)
	jtesting.AssertNoError(t, err)

	// 1 governance + 1 orders + 1 project + 2 skills (SKILL.md each) + 1 agent = 6 copies.
	jtesting.AssertEqual(t, len(copies), 6)

	// Governance file.
	jtesting.AssertEqual(t, strings.HasSuffix(copies[0].src, "governance/PHILOSOPHY.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[0].dst, ".claude/PHILOSOPHY.md"), true)

	// Orders file.
	jtesting.AssertEqual(t, strings.HasSuffix(copies[1].src, "teams/blue/ORDERS.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[1].dst, ".claude/ORDERS.md"), true)

	// Project file.
	jtesting.AssertEqual(t, strings.HasSuffix(copies[2].src, "projects/vicky/MISSION.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[2].dst, ".claude/MISSION.md"), true)

	// Skills (directory-based: teams/blue/skills/<name>/SKILL.md).
	jtesting.AssertEqual(t, strings.HasSuffix(copies[3].src, "skills/commit/SKILL.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[3].dst, ".claude/commands/commit/SKILL.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[4].src, "skills/pr/SKILL.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[4].dst, ".claude/commands/pr/SKILL.md"), true)

	// Agent (from teams/blue/agents/).
	jtesting.AssertEqual(t, strings.HasSuffix(copies[5].src, "teams/blue/agents/zidgel.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(copies[5].dst, ".claude/agents/zidgel.md"), true)
}

func TestApplyTeamNoAgentsDir(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/tmp/jack"}

	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("be good"), 0o600)

	// Team skills directory with one skill.
	teamSkillsDir := filepath.Join(configDir, "teams", "solo", "skills")
	_ = os.MkdirAll(filepath.Join(teamSkillsDir, "commit"), 0o750)
	_ = os.WriteFile(filepath.Join(teamSkillsDir, "commit", "SKILL.md"), []byte("commit"), 0o600)

	// Team ORDERS.md.
	_ = os.WriteFile(filepath.Join(configDir, "teams", "solo", "ORDERS.md"), []byte("orders"), 0o600)

	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("mission"), 0o600)

	var copyCount int
	copier := func(_, _ string) error {
		copyCount++
		return nil
	}

	dir := t.TempDir()
	err := applyTeam("solo", "vicky", dir, copier)
	jtesting.AssertNoError(t, err)
	// 1 governance + 1 orders + 1 project + 1 skill (SKILL.md) + 0 agents = 4.
	jtesting.AssertEqual(t, copyCount, 4)
}

func TestValidateGovernanceMissingDir(t *testing.T) {
	configDir := t.TempDir()
	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "governance directory"), true)
}

func TestValidateGovernanceEmptyDir(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "governance directory"), true)
}

func TestValidateGovernanceMissingTeamSkillsDir(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "team skills directory not found"), true)
}

func TestValidateGovernanceMissingOrdersMd(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue", "skills"), 0o750)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "ORDERS.md not found"), true)
}

func TestValidateGovernanceMissingProjectDir(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue", "skills"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "ORDERS.md"), []byte("x"), 0o600)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "project directory not found"), true)
}

func TestValidateGovernanceMissingMission(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue", "skills"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "ORDERS.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "MISSION.md not found"), true)
}

func TestValidateGovernanceSuccess(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue", "skills"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "ORDERS.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("x"), 0o600)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertNoError(t, err)
}
