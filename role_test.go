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
	err := applyTeam("bogus", "vicky", "/tmp/repo", noopLinker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "team skills directory"), true)
}

func TestApplyTeamLinksAllCategories(t *testing.T) {
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

	// Team files.
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "ORDERS.md"), []byte("orders"), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "PROTOCOL.md"), []byte("protocol"), 0o600)

	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("mission"), 0o600)

	// Team agent directory.
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue", "agents"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "blue", "agents", "zidgel.md"), []byte("zidgel"), 0o600)

	type linkOp struct{ src, dst string }
	var links []linkOp

	linker := func(src, dst string) error {
		links = append(links, linkOp{src, dst})
		return nil
	}

	dir := t.TempDir()
	err := applyTeam("blue", "vicky", dir, linker)
	jtesting.AssertNoError(t, err)

	// 1 governance + 2 team files + 1 project + 0 skills (symlinked directly) + 1 agent = 5 links.
	jtesting.AssertEqual(t, len(links), 5)

	// Governance file.
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].src, "governance/PHILOSOPHY.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].dst, ".claude/PHILOSOPHY.md"), true)

	// Team files (alphabetical: ORDERS.md, PROTOCOL.md).
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].src, "teams/blue/ORDERS.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].dst, ".claude/ORDERS.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[2].src, "teams/blue/PROTOCOL.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[2].dst, ".claude/PROTOCOL.md"), true)

	// Project file.
	jtesting.AssertEqual(t, strings.HasSuffix(links[3].src, "projects/vicky/MISSION.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[3].dst, ".claude/MISSION.md"), true)

	// Skills — verify symlinks created in .claude/commands/.
	commitLink := filepath.Join(dir, ".claude", "commands", "commit")
	target, err := os.Readlink(commitLink)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, target, filepath.Join(teamSkillsDir, "commit"))

	prLink := filepath.Join(dir, ".claude", "commands", "pr")
	target, err = os.Readlink(prLink)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, target, filepath.Join(teamSkillsDir, "pr"))

	// Agent (from teams/blue/agents/).
	jtesting.AssertEqual(t, strings.HasSuffix(links[4].src, "teams/blue/agents/zidgel.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[4].dst, ".claude/agents/zidgel.md"), true)
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

	// Team files.
	_ = os.WriteFile(filepath.Join(configDir, "teams", "solo", "ORDERS.md"), []byte("orders"), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "teams", "solo", "PROTOCOL.md"), []byte("protocol"), 0o600)

	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("mission"), 0o600)

	var linkCount int
	linker := func(_, _ string) error {
		linkCount++
		return nil
	}

	dir := t.TempDir()
	err := applyTeam("solo", "vicky", dir, linker)
	jtesting.AssertNoError(t, err)
	// 1 governance + 2 team files + 1 project + 0 skills (symlinked) + 0 agents = 4.
	jtesting.AssertEqual(t, linkCount, 4)
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
