//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestApplyRoleUnknownRole(t *testing.T) {
	newTestConfig()
	err := applyRole("bogus", "blue", "/tmp/repo", noopCopier)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown role"), true)
}

func TestApplyRoleUnknownTeam(t *testing.T) {
	newTestConfig()
	err := applyRole("developer", "bogus", "/tmp/repo", noopCopier)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown team"), true)
}

func TestApplyRoleCopiesSkillsAndAgents(t *testing.T) {
	newTestConfig()
	env = Env{ConfigDir: "/etc/jack", DataDir: "/tmp/jack"}

	type copyOp struct{ src, dst string }
	var copies []copyOp

	copier := func(src, dst string) error {
		copies = append(copies, copyOp{src, dst})
		return nil
	}

	dir := t.TempDir()
	err := applyRole("developer", "blue", dir, copier)
	jtesting.AssertNoError(t, err)

	// 2 skills (commit, pr) + 1 agent (zidgel) = 3 copies.
	jtesting.AssertEqual(t, len(copies), 3)

	// Skills copied from config dir to .claude/commands/.
	jtesting.AssertEqual(t, copies[0].src, "/etc/jack/skills/commit.md")
	jtesting.AssertEqual(t, strings.HasSuffix(copies[0].dst, ".claude/commands/commit.md"), true)
	jtesting.AssertEqual(t, copies[1].src, "/etc/jack/skills/pr.md")
	jtesting.AssertEqual(t, strings.HasSuffix(copies[1].dst, ".claude/commands/pr.md"), true)

	// Agent copied from config dir to .claude/agents/.
	jtesting.AssertEqual(t, copies[2].src, "/etc/jack/agents/zidgel.md")
	jtesting.AssertEqual(t, strings.HasSuffix(copies[2].dst, ".claude/agents/zidgel.md"), true)
}

func TestApplyRoleNoAgents(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"rockhopper": {},
		},
		Roles: map[string]Role{
			"minimal": {Skills: []string{"commit"}},
		},
		Teams: map[string]Team{
			"solo": {Profile: "rockhopper"},
		},
	}
	env = Env{ConfigDir: "/etc/jack", DataDir: "/tmp/jack"}

	var copyCount int
	copier := func(_, _ string) error {
		copyCount++
		return nil
	}

	dir := t.TempDir()
	err := applyRole("minimal", "solo", dir, copier)
	jtesting.AssertNoError(t, err)
	// Only 1 skill, no agents.
	jtesting.AssertEqual(t, copyCount, 1)
}
