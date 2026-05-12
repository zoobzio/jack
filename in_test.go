//go:build testing

package jack

import (
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
	err := runIn("", "", stubRegistry(), failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "no projects cloned"), true)
}

func TestRunInNoProjectsForAgent(t *testing.T) {
	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})
	err := runIn("red", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper)
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

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		existsChecker, noopCreator, attacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper)
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

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, creator, attacher,
		runner, noopContainerExecer, noopContainerStopper)
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

	err := runIn("", "", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, attacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, attachedName, "blue-vicky")
}

func TestRunInUnknownAgentProfile(t *testing.T) {
	newTestConfig()
	env = Env{DataDir: t.TempDir(), ConfigDir: t.TempDir()}

	reg := stubRegistry(RegistryEntry{Agent: "unknown", Repo: "vicky"})

	err := runIn("unknown", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, noopContainerExecer, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "unknown agent"), true)
}

func TestSetupScripts(t *testing.T) {
	configDir := t.TempDir()
	env = Env{ConfigDir: configDir}

	scripts := setupScripts("blue", "vicky")
	jtesting.AssertEqual(t, len(scripts), 3)

	// Global setup.
	jtesting.AssertEqual(t, scripts[0].label, "global setup")
	jtesting.AssertEqual(t, strings.HasSuffix(scripts[0].hostPath, "setup.sh"), true)
	jtesting.AssertEqual(t, scripts[0].containerPath, "/home/jack/.config/jack/setup.sh")

	// Agent setup.
	jtesting.AssertEqual(t, scripts[1].label, "agent setup for blue")
	jtesting.AssertEqual(t, strings.Contains(scripts[1].hostPath, "agents/blue/setup.sh"), true)
	jtesting.AssertEqual(t, scripts[1].containerPath, "/home/jack/.config/jack/agents/blue/setup.sh")

	// Project dev.
	jtesting.AssertEqual(t, scripts[2].label, "project setup for vicky")
	jtesting.AssertEqual(t, strings.Contains(scripts[2].hostPath, "projects/vicky/dev.sh"), true)
	jtesting.AssertEqual(t, scripts[2].containerPath, "/home/jack/.config/jack/projects/vicky/dev.sh")
}

func TestRunInExecutesSetupScripts(t *testing.T) {
	newTestConfig()
	configDir := t.TempDir()
	env = Env{DataDir: t.TempDir(), ConfigDir: configDir}

	// Create the global setup script so it gets executed.
	_ = os.WriteFile(filepath.Join(configDir, "setup.sh"), []byte("echo global"), 0o600)

	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	var execedCmds []string
	execer := func(_ string, cmd []string) error {
		execedCmds = append(execedCmds, strings.Join(cmd, " "))
		return nil
	}

	err := runIn("blue", "vicky", reg, failSelector, failProjectSelector,
		noopChecker, noopCreator, noopAttacher,
		noopContainerRunner, execer, noopContainerStopper)
	jtesting.AssertNoError(t, err)

	// Only global setup exists, so only one exec call for setup.
	jtesting.AssertEqual(t, len(execedCmds), 1)
	jtesting.AssertEqual(t, strings.Contains(execedCmds[0], "setup.sh"), true)
}
