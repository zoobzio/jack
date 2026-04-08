//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestApplyAgentUnknownAgent(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/tmp/jack"}
	err := applyAgent("bogus", "vicky", "/tmp/repo", noopLinker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "agent skills directory"), true)
}

func TestApplyAgentLinksAllCategories(t *testing.T) {
	// Set up a config directory with governance, agent skills, orders, projects.
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/tmp/jack"}

	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("be good"), 0o600)

	// Agent skills directory with skill dirs.
	agentSkillsDir := filepath.Join(configDir, "agents", "blue", "skills")
	_ = os.MkdirAll(filepath.Join(agentSkillsDir, "commit"), 0o750)
	_ = os.WriteFile(filepath.Join(agentSkillsDir, "commit", "SKILL.md"), []byte("commit"), 0o600)
	_ = os.MkdirAll(filepath.Join(agentSkillsDir, "pr"), 0o750)
	_ = os.WriteFile(filepath.Join(agentSkillsDir, "pr", "SKILL.md"), []byte("pr"), 0o600)

	// Agent files.
	_ = os.WriteFile(filepath.Join(configDir, "agents", "blue", "ORDERS.md"), []byte("orders"), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "agents", "blue", "PROTOCOL.md"), []byte("protocol"), 0o600)

	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("mission"), 0o600)

	type linkOp struct{ src, dst string }
	var links []linkOp

	linker := func(src, dst string) error {
		links = append(links, linkOp{src, dst})
		return nil
	}

	dir := t.TempDir()
	err := applyAgent("blue", "vicky", dir, linker)
	jtesting.AssertNoError(t, err)

	// 1 governance + 2 agent files + 1 project + 0 skills (symlinked directly) = 4 links.
	jtesting.AssertEqual(t, len(links), 4)

	// Governance file.
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].src, "governance/PHILOSOPHY.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].dst, ".claude/PHILOSOPHY.md"), true)

	// Agent files (alphabetical: ORDERS.md, PROTOCOL.md).
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].src, "agents/blue/ORDERS.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].dst, ".claude/ORDERS.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[2].src, "agents/blue/PROTOCOL.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[2].dst, ".claude/PROTOCOL.md"), true)

	// Project file.
	jtesting.AssertEqual(t, strings.HasSuffix(links[3].src, "projects/vicky/MISSION.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[3].dst, ".claude/MISSION.md"), true)

	// Skills — verify symlinks created in .claude/commands/.
	commitLink := filepath.Join(dir, ".claude", "commands", "commit")
	target, err := os.Readlink(commitLink)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, target, filepath.Join(agentSkillsDir, "commit"))

	prLink := filepath.Join(dir, ".claude", "commands", "pr")
	target, err = os.Readlink(prLink)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, target, filepath.Join(agentSkillsDir, "pr"))
}

func TestApplyAgentNoAgentsDir(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/tmp/jack"}

	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("be good"), 0o600)

	// Agent skills directory with one skill.
	agentSkillsDir := filepath.Join(configDir, "agents", "solo", "skills")
	_ = os.MkdirAll(filepath.Join(agentSkillsDir, "commit"), 0o750)
	_ = os.WriteFile(filepath.Join(agentSkillsDir, "commit", "SKILL.md"), []byte("commit"), 0o600)

	// Agent files.
	_ = os.WriteFile(filepath.Join(configDir, "agents", "solo", "ORDERS.md"), []byte("orders"), 0o600)
	_ = os.WriteFile(filepath.Join(configDir, "agents", "solo", "PROTOCOL.md"), []byte("protocol"), 0o600)

	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("mission"), 0o600)

	var linkCount int
	linker := func(_, _ string) error {
		linkCount++
		return nil
	}

	dir := t.TempDir()
	err := applyAgent("solo", "vicky", dir, linker)
	jtesting.AssertNoError(t, err)
	// 1 governance + 2 agent files + 1 project + 0 skills (symlinked) + 0 agent defs = 4.
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

func TestValidateGovernanceMissingAgentSkillsDir(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "agent skills directory not found"), true)
}

func TestValidateGovernanceMissingProjectDir(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "agents", "blue", "skills"), 0o750)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "project directory not found"), true)
}

func TestValidateGovernanceMissingMission(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "agents", "blue", "skills"), 0o750)
	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "MISSION.md not found"), true)
}

func TestValidateGovernanceSuccess(t *testing.T) {
	configDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(configDir, "governance"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "governance", "PHILOSOPHY.md"), []byte("x"), 0o600)
	_ = os.MkdirAll(filepath.Join(configDir, "agents", "blue", "skills"), 0o750)
	_ = os.MkdirAll(filepath.Join(configDir, "projects", "vicky"), 0o750)
	_ = os.WriteFile(filepath.Join(configDir, "projects", "vicky", "MISSION.md"), []byte("x"), 0o600)

	err := validateGovernance(configDir, "blue", "vicky")
	jtesting.AssertNoError(t, err)
}
