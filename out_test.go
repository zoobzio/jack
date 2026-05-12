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
