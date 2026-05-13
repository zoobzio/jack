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
	err := applyAgent("bogus")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "agent directory not found"), true)
}

func TestApplyAgentCopiesFiles(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	agentDir := filepath.Join(configDir, "agents", "blue")
	_ = os.MkdirAll(agentDir, 0o750)
	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("soul"), 0o600)
	_ = os.WriteFile(filepath.Join(agentDir, "settings.json"), []byte("{}"), 0o600)

	err := applyAgent("blue")
	jtesting.AssertNoError(t, err)

	claudeDir := filepath.Join(dataDir, "blue", ".claude")
	data, err := os.ReadFile(filepath.Join(claudeDir, "CLAUDE.md"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, string(data), "soul")

	data, err = os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, string(data), "{}")

	// Verify they are real files, not symlinks.
	info, err := os.Lstat(filepath.Join(claudeDir, "CLAUDE.md"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, info.Mode().IsRegular(), true)
}

func TestApplyAgentCopiesSubdirectories(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	agentDir := filepath.Join(configDir, "agents", "blue")
	commandsDir := filepath.Join(agentDir, "commands", "commit")
	_ = os.MkdirAll(commandsDir, 0o750)
	_ = os.WriteFile(filepath.Join(commandsDir, "SKILL.md"), []byte("commit skill"), 0o600)
	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("soul"), 0o600)

	err := applyAgent("blue")
	jtesting.AssertNoError(t, err)

	claudeDir := filepath.Join(dataDir, "blue", ".claude")
	data, err := os.ReadFile(filepath.Join(claudeDir, "commands", "commit", "SKILL.md"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, string(data), "commit skill")
}

func TestApplyAgentFollowsSymlinks(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	sharedDir := filepath.Join(configDir, "commands", "review")
	_ = os.MkdirAll(sharedDir, 0o750)
	_ = os.WriteFile(filepath.Join(sharedDir, "SKILL.md"), []byte("review skill"), 0o600)

	agentDir := filepath.Join(configDir, "agents", "blue")
	agentCmdsDir := filepath.Join(agentDir, "commands")
	_ = os.MkdirAll(agentCmdsDir, 0o750)
	_ = os.Symlink(sharedDir, filepath.Join(agentCmdsDir, "review"))
	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("soul"), 0o600)

	err := applyAgent("blue")
	jtesting.AssertNoError(t, err)

	claudeDir := filepath.Join(dataDir, "blue", ".claude")
	data, err := os.ReadFile(filepath.Join(claudeDir, "commands", "review", "SKILL.md"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, string(data), "review skill")
}

func TestApplyAgentEmptyDir(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	_ = os.MkdirAll(filepath.Join(configDir, "agents", "blue"), 0o750)

	err := applyAgent("blue")
	jtesting.AssertNoError(t, err)

	entries, err := os.ReadDir(filepath.Join(dataDir, "blue", ".claude"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(entries), 0)
}

func TestApplyAgentCreateClaudeDirError(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: "/dev/null/impossible"}

	_ = os.MkdirAll(filepath.Join(configDir, "agents", "blue"), 0o750)

	err := applyAgent("blue")
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "creating agent .claude dir"), true)
}

func TestCopyDirRecursiveReadDirError(t *testing.T) {
	err := copyDirRecursive("/nonexistent/dir", t.TempDir())
	jtesting.AssertError(t, err)
}

func TestCopyDirRecursiveStatError(t *testing.T) {
	srcDir := t.TempDir()
	// Create a broken symlink.
	_ = os.Symlink("/nonexistent/target", filepath.Join(srcDir, "broken"))

	err := copyDirRecursive(srcDir, t.TempDir())
	jtesting.AssertError(t, err)
}

func TestCopyDirRecursiveSubdirError(t *testing.T) {
	srcDir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(srcDir, "subdir"), 0o750)

	err := copyDirRecursive(srcDir, "/dev/null/impossible")
	jtesting.AssertError(t, err)
}

func TestCopyFileContentError(t *testing.T) {
	err := copyFileContent("/nonexistent/file", filepath.Join(t.TempDir(), "dst"))
	jtesting.AssertError(t, err)
}

func TestCopyFileContentCreateError(t *testing.T) {
	src := filepath.Join(t.TempDir(), "src")
	_ = os.WriteFile(src, []byte("data"), 0o600)

	err := copyFileContent(src, "/dev/null/impossible/dst")
	jtesting.AssertError(t, err)
}

func TestApplyAgentOverwritesPrevious(t *testing.T) {
	configDir := t.TempDir()
	dataDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: dataDir}

	agentDir := filepath.Join(configDir, "agents", "blue")
	_ = os.MkdirAll(agentDir, 0o750)
	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("v1"), 0o600)

	_ = applyAgent("blue")

	_ = os.WriteFile(filepath.Join(agentDir, "CLAUDE.md"), []byte("v2"), 0o600)
	_ = applyAgent("blue")

	data, err := os.ReadFile(filepath.Join(dataDir, "blue", ".claude", "CLAUDE.md"))
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, string(data), "v2")
}
