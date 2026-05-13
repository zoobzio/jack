//go:build testing

package jack

import (
	"fmt"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunOutNotFound(t *testing.T) {
	err := runOut("blue-vicky", "", "", "", noopChecker, noopKiller, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "not found"), true)
}

func TestRunOutSuccess(t *testing.T) {
	var killed string
	err := runOut("blue-vicky", "", "", "", existsChecker, func(name string) error {
		killed = name
		return nil
	}, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, killed, "blue-vicky")
}

func TestRunOutWithFlags(t *testing.T) {
	var killed string
	err := runOut("", "blue", "vicky", "", existsChecker, func(name string) error {
		killed = name
		return nil
	}, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, killed, "blue-vicky")
}

func TestRunOutStopsContainer(t *testing.T) {
	var stoppedContainer string
	stopper := func(name string) error {
		stoppedContainer = name
		return nil
	}
	err := runOut("blue-vicky", "", "", "", existsChecker, noopKiller, stopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, stoppedContainer, "jack-blue-vicky")
}

func TestRunOutWorktreeDoesNotStopContainer(t *testing.T) {
	var containerStopped bool
	stopper := func(_ string) error {
		containerStopped = true
		return nil
	}
	err := runOut("", "blue", "vicky", "feature-auth", existsChecker, noopKiller, stopper)
	jtesting.AssertNoError(t, err)
	// Worktree sessions share the container — should NOT stop it.
	jtesting.AssertEqual(t, containerStopped, false)
}

func TestRunOutKillError(t *testing.T) {
	failKiller := func(_ string) error { return fmt.Errorf("kill failed") }
	err := runOut("blue-vicky", "", "", "", existsChecker, failKiller, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "kill failed"), true)
}

func TestRunOutMissingArgs(t *testing.T) {
	err := runOut("", "", "", "", noopChecker, noopKiller, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "specify a session name"), true)
}

func TestParseSessionName(t *testing.T) {
	agent, project := parseSessionName("blue-vicky")
	jtesting.AssertEqual(t, agent, "blue")
	jtesting.AssertEqual(t, project, "vicky")

	agent, project = parseSessionName("rockhopper-sentinel")
	jtesting.AssertEqual(t, agent, "rockhopper")
	jtesting.AssertEqual(t, project, "sentinel")
}

func TestParseSessionNameNoHyphen(t *testing.T) {
	agent, project := parseSessionName("singleword")
	jtesting.AssertEqual(t, agent, "singleword")
	jtesting.AssertEqual(t, project, "")
}

func TestRunOutNilStopper(t *testing.T) {
	// When stopContainer is nil, should still succeed without panicking.
	err := runOut("blue-vicky", "", "", "", existsChecker, noopKiller, nil)
	jtesting.AssertNoError(t, err)
}

func TestRunOutSingleWordSessionName(t *testing.T) {
	// Session name with no hyphen — parseSessionName returns empty project.
	// The container stop should be skipped since project is empty.
	var containerStopped bool
	stopper := func(_ string) error {
		containerStopped = true
		return nil
	}
	err := runOut("singleword", "", "", "", existsChecker, noopKiller, stopper)
	jtesting.AssertNoError(t, err)
	// project is empty from parseSessionName, so stopContainer should NOT be called.
	jtesting.AssertEqual(t, containerStopped, false)
}

func TestRunOutContainerStopError(t *testing.T) {
	stopper := func(_ string) error {
		return fmt.Errorf("container not found")
	}
	// Should succeed despite container stop error (non-fatal warning).
	err := runOut("blue-vicky", "", "", "", existsChecker, noopKiller, stopper)
	jtesting.AssertNoError(t, err)
}

func TestRunOutAll(t *testing.T) {
	reg := stubRegistry(
		RegistryEntry{Agent: "blue", Repo: "vicky"},
		RegistryEntry{Agent: "blue", Repo: "flux"},
		RegistryEntry{Agent: "red", Repo: "sentinel"},
	)

	// Only blue-vicky and red-sentinel have active tmux sessions.
	sessions := func() ([]TmuxSession, error) {
		return []TmuxSession{
			{Name: "blue-vicky"},
			{Name: "red-sentinel"},
			{Name: "personal"}, // not managed by jack
		}, nil
	}

	hasSession := func(name string) bool {
		return name == "blue-vicky" || name == "red-sentinel"
	}

	var killed []string
	killer := func(name string) error {
		killed = append(killed, name)
		return nil
	}

	err := runOutAll(reg, sessions, hasSession, killer, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(killed), 2)
}

func TestRunOutAllNoSessions(t *testing.T) {
	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	sessions := func() ([]TmuxSession, error) {
		return nil, nil
	}

	var killed int
	killer := func(_ string) error {
		killed++
		return nil
	}

	err := runOutAll(reg, sessions, noopChecker, killer, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, killed, 0)
}

func TestRunOutAllRegistryError(t *testing.T) {
	failReg := func() (*Registry, error) { return nil, fmt.Errorf("corrupt") }

	err := runOutAll(failReg, nil, noopChecker, noopKiller, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "loading registry"), true)
}

func TestRunOutAllSkipsNonManaged(t *testing.T) {
	reg := stubRegistry(RegistryEntry{Agent: "blue", Repo: "vicky"})

	// Only a non-managed session exists.
	sessions := func() ([]TmuxSession, error) {
		return []TmuxSession{{Name: "personal"}}, nil
	}

	var killed int
	killer := func(_ string) error {
		killed++
		return nil
	}

	err := runOutAll(reg, sessions, existsChecker, killer, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, killed, 0)
}
