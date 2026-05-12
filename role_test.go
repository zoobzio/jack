//go:build testing

package jack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestApplyAgentMissingDir(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	err := applyAgent("bogus", noopLinker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "agent directory not found"), true)
}

func TestApplyAgentLinksFiles(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	// Create agent config directory with files.
	agentDir := filepath.Join(configDir, "agents", "blue")
	_ = os.MkdirAll(agentDir, 0o750)
	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("soul"), 0o600)
	_ = os.WriteFile(filepath.Join(agentDir, "settings.json"), []byte("{}"), 0o600)

	type linkOp struct{ src, dst string }
	var links []linkOp

	linker := func(src, dst string) error {
		links = append(links, linkOp{src, dst})
		return nil
	}

	err := applyAgent("blue", linker)
	jtesting.AssertNoError(t, err)

	jtesting.AssertEqual(t, len(links), 2)
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].src, "agents/blue/CLAUDE.md"), true)
	// Target should be in the agent's data dir .claude/
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].dst, ".claude/CLAUDE.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].src, "agents/blue/settings.json"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].dst, ".claude/settings.json"), true)
}

func TestApplyAgentLinksSubdirectories(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	// Create agent config with a commands/ subdirectory.
	agentDir := filepath.Join(configDir, "agents", "blue")
	commandsDir := filepath.Join(agentDir, "commands", "commit")
	_ = os.MkdirAll(commandsDir, 0o750)
	_ = os.WriteFile(filepath.Join(commandsDir, "SKILL.md"), []byte("commit skill"), 0o600)
	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("soul"), 0o600)

	type linkOp struct{ src, dst string }
	var links []linkOp

	linker := func(src, dst string) error {
		links = append(links, linkOp{src, dst})
		return nil
	}

	err := applyAgent("blue", linker)
	jtesting.AssertNoError(t, err)

	// 1 top-level file + 1 file inside commands/commit/
	jtesting.AssertEqual(t, len(links), 2)
	jtesting.AssertEqual(t, strings.HasSuffix(links[0].src, "CLAUDE.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].src, "SKILL.md"), true)
	jtesting.AssertEqual(t, strings.HasSuffix(links[1].dst, ".claude/commands/commit/SKILL.md"), true)
}

func TestApplyAgentEmptyDir(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	_ = os.MkdirAll(filepath.Join(configDir, "agents", "blue"), 0o750)

	var linkCount int
	linker := func(_, _ string) error {
		linkCount++
		return nil
	}

	err := applyAgent("blue", linker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, linkCount, 0)
}
