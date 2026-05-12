//go:build testing

package jack

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

// setupWorktreeFixtures creates .git/worktrees/{name}/HEAD files for testing.
func setupWorktreeFixtures(t *testing.T, agent, repo string, branches map[string]string) {
	t.Helper()
	dataDir := env.dataDir()
	for name, branch := range branches {
		wtDir := filepath.Join(dataDir, agent, repo, ".git", "worktrees", name)
		err := os.MkdirAll(wtDir, 0o750)
		jtesting.AssertNoError(t, err)
		err = os.WriteFile(
			filepath.Join(wtDir, "HEAD"),
			[]byte(fmt.Sprintf("ref: refs/heads/%s\n", branch)),
			0o600,
		)
		jtesting.AssertNoError(t, err)
	}
}

func TestRunPruneNoWorktrees(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	// No worktree directories at all — listWorktreeBranches returns nil.
	err := runPrune("blue", "vicky", true, false,
		noopChecker, noopContainerChecker, noopContainerExecer)
	jtesting.AssertNoError(t, err)
}

func TestRunPruneNoUnused(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	setupWorktreeFixtures(t, "blue", "vicky", map[string]string{
		"wt1": "feature-1",
		"wt2": "feature-2",
	})

	// All worktrees have active sessions.
	allActive := func(_ string) bool { return true }

	err := runPrune("blue", "vicky", true, false,
		allActive, noopContainerChecker, noopContainerExecer)
	jtesting.AssertNoError(t, err)
}

func TestRunPruneDryRun(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	setupWorktreeFixtures(t, "blue", "vicky", map[string]string{
		"wt1": "feature-1",
		"wt2": "feature-2",
	})

	// No sessions active — both are prunable.
	var execCalled bool
	trackExec := func(_ string, _ []string) error {
		execCalled = true
		return nil
	}

	err := runPrune("blue", "vicky", true, true,
		noopChecker, noopContainerChecker, trackExec)
	jtesting.AssertNoError(t, err)
	// Dry run should not exec anything.
	jtesting.AssertEqual(t, execCalled, false)
}

func TestRunPruneContainerNotRunning(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	setupWorktreeFixtures(t, "blue", "vicky", map[string]string{
		"wt1": "feature-1",
	})

	// Container is not running.
	notRunning := func(_ string) (bool, bool) { return false, false }

	err := runPrune("blue", "vicky", true, false,
		noopChecker, notRunning, noopContainerExecer)
	jtesting.AssertError(t, err)
}

func TestRunPruneSuccessful(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	setupWorktreeFixtures(t, "blue", "vicky", map[string]string{
		"wt1": "feature-1",
	})

	// Container is running.
	running := func(_ string) (bool, bool) { return true, true }

	var execCmds [][]string
	trackExec := func(name string, cmd []string) error {
		execCmds = append(execCmds, cmd)
		return nil
	}

	err := runPrune("blue", "vicky", true, false,
		noopChecker, running, trackExec)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(execCmds), 1)

	// Verify the command includes "git worktree remove".
	cmd := execCmds[0]
	jtesting.AssertEqual(t, cmd[0], "git")
	jtesting.AssertEqual(t, cmd[3], "worktree")
	jtesting.AssertEqual(t, cmd[4], "remove")
}

func TestRunPruneExecError(t *testing.T) {
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir}

	setupWorktreeFixtures(t, "blue", "vicky", map[string]string{
		"wt1": "feature-1",
		"wt2": "feature-2",
	})

	// Container is running.
	running := func(_ string) (bool, bool) { return true, true }

	// execContainer fails — should print warning and continue.
	failExec := func(_ string, _ []string) error {
		return fmt.Errorf("worktree remove failed")
	}

	err := runPrune("blue", "vicky", true, false,
		noopChecker, running, failExec)
	jtesting.AssertNoError(t, err)
}
