//go:build testing

package jack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func stubRegistry(entries ...RegistryEntry) RegistryLoader {
	return func() (*Registry, error) {
		return &Registry{Projects: entries}, nil
	}
}

func stubAgentSelector(agent string) AgentSelector {
	return func(_ []string) (string, error) { return agent, nil }
}

func stubProjectSelector(project string) ProjectSelector {
	return func(_ string, _ []string) (string, error) { return project, nil }
}

var failSelector AgentSelector = func(_ []string) (string, error) {
	return "", nil
}

var failProjectSelector ProjectSelector = func(_ string, _ []string) (string, error) {
	return "", nil
}

func TestRunInEmptyRegistry(t *testing.T) {
	err := runIn("", "", "", stubRegistry(), failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "no projects cloned"), true)
}

func TestRunInNoProjectsForAgent(t *testing.T) {
	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})
	err := runIn("red", "", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "no projects cloned for agent"), true)
}

func TestRunInAttachesExistingSession(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var attachedName string
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		existsChecker, noopCreator, attacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInCreatesContainerAndSession(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var createdName, attachedName string
	var containerStarted string
	creator := func(name, dir, shellCmd string) error {
		createdName = name
		return nil
	}
	attacher := func(name string) error {
		attachedName = name
		return nil
	}
	runner := func(name string, _ []Mount, _ []Volume, _ map[string]string) error {
		containerStarted = name
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, creator, attacher,
		runner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, createdName, "blue-vicky")
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
	jtesting.AssertEqual(t, containerStarted, "jack-blue-vicky")
}

func TestRunInAutoSelectsSingleAgent(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var attachedName string
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	err := runIn("", "", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, attacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInUnknownAgentProfile(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "unknown", Repo: "vicky"})

	err := runIn("unknown", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown agent"), true)
}

func TestRunInWithWorktreeUsesHash(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var createdName string
	creator := func(name, dir, shellCmd string) error {
		createdName = name
		return nil
	}

	// Container already running.
	containerRunning := func(_ string) (bool, bool) { return true, true }

	err := runIn("blue", "vicky", "feature-auth", reg, failSelector, failProjectSelector,
		noopChecker, creator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, containerRunning)
	jtesting.AssertNoError(t, err)

	hash := WorktreeHash("feature-auth")
	jtesting.AssertEqual(t, createdName, "blue-vicky-"+hash)
}

func TestRunInWorktreeMainBranchFallback(t *testing.T) {
	newTestConfig()
	dataDir := t.TempDir()
	env = Env{DataDir: dataDir, ConfigDir: t.TempDir()}

	// Create a fake .git/HEAD pointing to main.
	repoDir := filepath.Join(dataDir, "blue", "vicky")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0o750)
	_ = os.WriteFile(filepath.Join(repoDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var createdName string
	creator := func(name, dir, shellCmd string) error {
		createdName = name
		return nil
	}

	err := runIn("blue", "vicky", "main", reg, failSelector, failProjectSelector,
		noopChecker, creator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)

	// Should fall back to main session name (no hash suffix).
	jtesting.AssertEqual(t, createdName, "blue-vicky")
}

func TestReadHEADBranch(t *testing.T) {
	dir := t.TempDir()
	gitDir := filepath.Join(dir, ".git")
	_ = os.MkdirAll(gitDir, 0o750)

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o600)
	jtesting.AssertEqual(t, readHEADBranch(dir), "main")

	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/feature/auth\n"), 0o600)
	jtesting.AssertEqual(t, readHEADBranch(dir), "feature/auth")

	// Detached HEAD.
	_ = os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("abc123def456\n"), 0o600)
	jtesting.AssertEqual(t, readHEADBranch(dir), "")

	// Missing .git.
	jtesting.AssertEqual(t, readHEADBranch("/nonexistent"), "")
}

func TestSetupScripts(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir}

	scripts := setupScripts("blue", "vicky")
	jtesting.AssertEqual(t, len(scripts), 3)

	jtesting.AssertEqual(t, scripts[0].label, "global setup")
	jtesting.AssertEqual(t, scripts[0].containerPath, "/home/jack/.config/jack/setup.sh")

	jtesting.AssertEqual(t, scripts[1].label, "agent setup for blue")
	jtesting.AssertEqual(t, scripts[1].containerPath, "/home/jack/.config/jack/agents/blue/setup.sh")

	jtesting.AssertEqual(t, scripts[2].label, "project setup for vicky")
	jtesting.AssertEqual(t, scripts[2].containerPath, "/home/jack/.config/jack/projects/vicky/dev.sh")
}

func TestRunInRegistryLoadError(t *testing.T) {
	failReg := func() (*Registry, error) { return nil, fmt.Errorf("corrupt") }
	err := runIn("blue", "vicky", "", failReg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "loading registry"), true)
}

func TestRunInContainerStartError(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	failRunner := func(_ string, _ []Mount, _ []Volume, _ map[string]string) error {
		return fmt.Errorf("docker not available")
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		failRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "starting container"), true)
}

func TestRunInContainerAlreadyRunning(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var containerStarted bool
	runner := func(_ string, _ []Mount, _ []Volume, _ map[string]string) error {
		containerStarted = true
		return nil
	}

	// Container is already running.
	containerRunning := func(_ string) (bool, bool) { return true, true }

	var createdName string
	creator := func(name, dir, shellCmd string) error {
		createdName = name
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, creator, noopAttacher,
		runner, noopContainerExecer, noopContainerStopper, containerRunning)
	jtesting.AssertNoError(t, err)
	// Container was already running, so runner should NOT have been called.
	jtesting.AssertEqual(t, containerStarted, false)
	jtesting.AssertEqual(t, createdName, "blue-vicky")
}

func TestRunInWorktreeCreationError(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	// Container already running.
	containerRunning := func(_ string) (bool, bool) { return true, true }

	// execContainer fails for worktree creation.
	failExecer := func(_ string, _ []string) error {
		return fmt.Errorf("git worktree add failed")
	}

	err := runIn("blue", "vicky", "feature-broken", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, failExecer, noopContainerStopper, containerRunning)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "creating worktree"), true)
}

func TestRunInPromptsForAgentWithMultipleAgents(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {Git: GitConfig{Name: "Rockhopper", Email: "rock@example.com"}},
			"red":  {Git: GitConfig{Name: "Mother", Email: "mother@example.com"}},
		},
	}
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "red", Repo: "flux"},
	)

	var selectedFrom []string
	selAgent := func(agents []string) (string, error) {
		selectedFrom = agents
		return "blue", nil
	}

	var attachedName string
	attacher := func(name string) error {
		attachedName = name
		return nil
	}

	err := runIn("", "vicky", "", reg, selAgent, failProjectSelector,
		existsChecker, noopCreator, attacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(selectedFrom), 2)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInExecutesSetupScripts(t *testing.T) {
	newTestConfig()
	configDir := t.TempDir()
	env = Env{DataDir: t.TempDir(), ConfigDir: configDir}

	_ = os.WriteFile(filepath.Join(configDir, "setup.sh"), []byte("echo global"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var execedCmds []string
	execer := func(_ string, cmd []string) error {
		execedCmds = append(execedCmds, strings.Join(cmd, " "))
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, execer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)

	jtesting.AssertEqual(t, len(execedCmds), 1)
	jtesting.AssertEqual(t, strings.Contains(execedCmds[0], "setup.sh"), true)
}

func TestRunInSetupScriptError(t *testing.T) {
	newTestConfig()
	configDir := t.TempDir()
	env = Env{DataDir: t.TempDir(), ConfigDir: configDir}

	_ = os.WriteFile(filepath.Join(configDir, "setup.sh"), []byte("echo global"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	failExecer := func(_ string, _ []string) error {
		return fmt.Errorf("setup failed")
	}

	var containerStopped bool
	stopper := func(_ string) error {
		containerStopped = true
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, failExecer, stopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "running"), true)
	jtesting.AssertEqual(t, containerStopped, true)
}

func TestRunInCreateSessionError(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	failCreator := func(_, _, _ string) error {
		return fmt.Errorf("tmux failed")
	}

	var containerStopped bool
	stopper := func(_ string) error {
		containerStopped = true
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, failCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, stopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "tmux failed"), true)
	// Container was newly started, so it should be stopped on error.
	jtesting.AssertEqual(t, containerStopped, true)
}

func TestRunInCreateSessionErrorContainerAlreadyRunning(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	failCreator := func(_, _, _ string) error {
		return fmt.Errorf("tmux failed")
	}

	containerRunning := func(_ string) (bool, bool) { return true, true }

	var containerStopped bool
	stopper := func(_ string) error {
		containerStopped = true
		return nil
	}

	err := runIn("blue", "vicky", "", reg, failSelector, failProjectSelector,
		noopChecker, failCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, stopper, containerRunning)
	jtesting.AssertError(t, err)
	// Container was already running, so should NOT be stopped.
	jtesting.AssertEqual(t, containerStopped, false)
}

func TestRunInAgentSelectorError(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {},
			"red":  {},
		},
	}
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "red", Repo: "flux"},
	)

	failAgent := func(_ []string) (string, error) {
		return "", fmt.Errorf("cancelled")
	}

	err := runIn("", "", "", reg, failAgent, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "cancelled"), true)
}

func TestRunInProjectSelectorError(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "blue", Repo: "flux"},
	)

	failProject := func(_ string, _ []string) (string, error) {
		return "", fmt.Errorf("cancelled")
	}

	err := runIn("blue", "", "", reg, failSelector, failProject,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "cancelled"), true)
}

func TestRunInMultipleProjects(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "blue", Repo: "flux"},
	)

	var selectedFrom []string
	selProject := func(_ string, repos []string) (string, error) {
		selectedFrom = repos
		return "vicky", nil
	}

	err := runIn("blue", "", "", reg, failSelector, selProject,
		existsChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper, noopContainerChecker)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(selectedFrom), 2)
}
