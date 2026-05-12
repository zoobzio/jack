//go:build testing

package jack

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunStatusEmptyRegistry(t *testing.T) {
	var buf bytes.Buffer
	err := runStatus(&buf, stubRegistry(), func() ([]TmuxSession, error) {
		return nil, nil
	}, noopContainerChecker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, strings.Contains(buf.String(), "no projects cloned"), true)
}

func TestRunStatusNoSessions(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "red", Repo: "flux"},
	)

	var buf bytes.Buffer
	err := runStatus(&buf, reg, func() ([]TmuxSession, error) {
		return nil, nil
	}, noopContainerChecker)
	jtesting.AssertNoError(t, err)

	output := buf.String()
	jtesting.AssertEqual(t, strings.Contains(output, "blue"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "red"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "not running"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "BRANCH"), true)
}

func TestRunStatusWithSessions(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "blue", Repo: "flux"},
		RegistryEntry{Agent: "red", Repo: "sentinel"},
	)

	var buf bytes.Buffer
	err := runStatus(&buf, reg, func() ([]TmuxSession, error) {
		return []TmuxSession{
			{
				Name:     "blue-vicky",
				Created:  time.Now().Add(-time.Hour),
				Activity: time.Now(),
				Path:     "/home/user/vicky",
				Attached: true,
				Windows:  1,
			},
			{
				Name:     "blue-flux",
				Created:  time.Now(),
				Activity: time.Now(),
				Path:     "/home/user/flux",
				Attached: false,
				Windows:  1,
			},
			{
				Name:     "personal",
				Created:  time.Now(),
				Activity: time.Now(),
				Path:     "/home/user",
				Attached: false,
				Windows:  1,
			},
		}, nil
	}, noopContainerChecker)
	jtesting.AssertNoError(t, err)

	output := buf.String()
	jtesting.AssertEqual(t, strings.Contains(output, "blue-vicky"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "blue-flux"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "attached"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "active"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "red"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "not running"), true)
	// Non-jack session filtered out.
	jtesting.AssertEqual(t, strings.Contains(output, "personal"), false)
	// Column headers present.
	jtesting.AssertEqual(t, strings.Contains(output, "CONTAINER"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "BRANCH"), true)
}

func TestListWorktreeBranches(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// Create a worktree fixture.
	wtDir := filepath.Join(dataDir, "blue", "vicky", ".git", "worktrees", "wt1")
	err := os.MkdirAll(wtDir, 0o750)
	jtesting.AssertNoError(t, err)
	err = os.WriteFile(
		filepath.Join(wtDir, "HEAD"),
		[]byte(fmt.Sprintf("ref: refs/heads/%s\n", "feature-1")),
		0o600,
	)
	jtesting.AssertNoError(t, err)

	result := listWorktreeBranches("blue", "vicky")
	jtesting.AssertEqual(t, len(result), 1)
	hash, ok := result["feature-1"]
	jtesting.AssertEqual(t, ok, true)
	jtesting.AssertEqual(t, hash, WorktreeHash("feature-1"))
}

func TestListWorktreeBranchesNonDirEntry(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// Create .git/worktrees/ directory with a file (not a directory).
	wtDir := filepath.Join(dataDir, "blue", "vicky", ".git", "worktrees")
	err := os.MkdirAll(wtDir, 0o750)
	jtesting.AssertNoError(t, err)
	// Add a file — should be skipped (not a directory).
	_ = os.WriteFile(filepath.Join(wtDir, "somefile"), []byte("data"), 0o600)

	result := listWorktreeBranches("blue", "vicky")
	jtesting.AssertEqual(t, len(result), 0)
}

func TestListWorktreeBranchesDetachedHEAD(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// Create a worktree with a detached HEAD (no ref: prefix).
	wtDir := filepath.Join(dataDir, "blue", "vicky", ".git", "worktrees", "wt1")
	err := os.MkdirAll(wtDir, 0o750)
	jtesting.AssertNoError(t, err)
	_ = os.WriteFile(filepath.Join(wtDir, "HEAD"), []byte("abc123def456\n"), 0o600)

	result := listWorktreeBranches("blue", "vicky")
	jtesting.AssertEqual(t, len(result), 0)
}

func TestListWorktreeBranchesUnreadableHEAD(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// Create a worktree directory without a HEAD file.
	wtDir := filepath.Join(dataDir, "blue", "vicky", ".git", "worktrees", "wt1")
	err := os.MkdirAll(wtDir, 0o750)
	jtesting.AssertNoError(t, err)
	// No HEAD file — should be skipped.

	result := listWorktreeBranches("blue", "vicky")
	jtesting.AssertEqual(t, len(result), 0)
}

func TestListWorktreeBranchesEmpty(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	result := listWorktreeBranches("blue", "nonexistent")
	jtesting.AssertEqual(t, len(result), 0)
}

func TestContainerStatus(t *testing.T) {
	jtesting.AssertEqual(t, containerStatus(true, true), "running")
	jtesting.AssertEqual(t, containerStatus(false, true), "stopped")
	jtesting.AssertEqual(t, containerStatus(false, false), "-")
}

func TestFormatDuration(t *testing.T) {
	t.Run("minutes", func(t *testing.T) {
		jtesting.AssertEqual(t, formatDuration(5*time.Minute), "5m")
	})
	t.Run("hours", func(t *testing.T) {
		jtesting.AssertEqual(t, formatDuration(3*time.Hour), "3h")
	})
	t.Run("days", func(t *testing.T) {
		jtesting.AssertEqual(t, formatDuration(48*time.Hour), "2d")
	})
}

func TestCertStatusLabelNoCert(t *testing.T) {
	// Agent with no cert file — hasCert returns false.
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	_ = os.MkdirAll(filepath.Join(tmpDir, "agents", "nocert"), 0o750)
	jtesting.AssertEqual(t, certStatusLabel("nocert"), "(no cert)")
}

func TestCertStatusLabelCertError(t *testing.T) {
	// Agent with a cert file that is not valid PEM.
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	agentDir := filepath.Join(tmpDir, "agents", "baderror")
	_ = os.MkdirAll(agentDir, 0o750)
	_ = os.WriteFile(filepath.Join(agentDir, "cert.pem"), []byte("not valid pem"), 0o600)
	jtesting.AssertEqual(t, certStatusLabel("baderror"), "(cert error)")
}

func TestCertStatusLabelCertExpired(t *testing.T) {
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	agentDir := filepath.Join(tmpDir, "agents", "expired")
	_ = os.MkdirAll(agentDir, 0o750)
	certFile := filepath.Join(agentDir, "cert.pem")
	// Cert that expired 1 hour ago.
	generateTestCert(t, certFile, time.Now().Add(-2*time.Hour), time.Now().Add(-1*time.Hour))
	jtesting.AssertEqual(t, certStatusLabel("expired"), "(cert expired)")
}

func TestCertStatusLabelCertExpiresIn(t *testing.T) {
	tmpDir := t.TempDir()
	env = Env{ConfigDir: tmpDir}
	agentDir := filepath.Join(tmpDir, "agents", "valid")
	_ = os.MkdirAll(agentDir, 0o750)
	certFile := filepath.Join(agentDir, "cert.pem")
	// Cert that expires in 2 hours.
	generateTestCert(t, certFile, time.Now().Add(-1*time.Hour), time.Now().Add(2*time.Hour))
	result := certStatusLabel("valid")
	jtesting.AssertEqual(t, strings.Contains(result, "cert expires in"), true)
	jtesting.AssertEqual(t, strings.Contains(result, "h"), true)
}

func TestSessionStatusIdle(t *testing.T) {
	info := SessionInfo{
		TmuxSession: TmuxSession{
			Attached: false,
			Activity: time.Now().Add(-5 * time.Minute),
		},
	}
	result := sessionStatus(info)
	jtesting.AssertEqual(t, strings.HasPrefix(result, "idle"), true)
	jtesting.AssertEqual(t, strings.Contains(result, "5m"), true)
}

func TestRunStatusWithWorktrees(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// Create a worktree fixture so listWorktreeBranches returns something.
	wtDir := filepath.Join(dataDir, "blue", "vicky", ".git", "worktrees", "wt1")
	_ = os.MkdirAll(wtDir, 0o750)
	_ = os.WriteFile(
		filepath.Join(wtDir, "HEAD"),
		[]byte(fmt.Sprintf("ref: refs/heads/%s\n", "feature-x")),
		0o600,
	)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	hash := WorktreeHash("feature-x")
	wtSessionName := "blue-vicky-" + hash

	var buf bytes.Buffer
	err := runStatus(&buf, reg, func() ([]TmuxSession, error) {
		return []TmuxSession{
			{
				Name:     "blue-vicky",
				Created:  time.Now(),
				Activity: time.Now(),
				Path:     "/tmp/vicky",
				Attached: false,
				Windows:  1,
			},
			{
				Name:     wtSessionName,
				Created:  time.Now(),
				Activity: time.Now(),
				Path:     "/tmp/vicky-wt",
				Attached: false,
				Windows:  1,
			},
		}, nil
	}, noopContainerChecker)
	jtesting.AssertNoError(t, err)

	output := buf.String()
	jtesting.AssertEqual(t, strings.Contains(output, "feature-x"), true)
	jtesting.AssertEqual(t, strings.Contains(output, wtSessionName), true)
}

func TestRunStatusWithWorktreeNotRunning(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// Create a worktree fixture.
	wtDir := filepath.Join(dataDir, "blue", "vicky", ".git", "worktrees", "wt1")
	_ = os.MkdirAll(wtDir, 0o750)
	_ = os.WriteFile(
		filepath.Join(wtDir, "HEAD"),
		[]byte("ref: refs/heads/feature-y\n"),
		0o600,
	)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var buf bytes.Buffer
	// No sessions running at all — both main and worktree should show "not running".
	err := runStatus(&buf, reg, func() ([]TmuxSession, error) {
		return nil, nil
	}, noopContainerChecker)
	jtesting.AssertNoError(t, err)

	output := buf.String()
	jtesting.AssertEqual(t, strings.Contains(output, "feature-y"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "not running"), true)
}
