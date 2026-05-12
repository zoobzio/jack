//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunOutNotFound(t *testing.T) {
	err := runOut("blue-vicky", "", "", noopChecker, noopKiller, noopContainerStopper)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "not found"), true)
}

func TestRunOutSuccess(t *testing.T) {
	var killed string
	err := runOut("blue-vicky", "", "", existsChecker, func(name string) error {
		killed = name
		return nil
	}, noopContainerStopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, killed, "blue-vicky")
}

func TestRunOutWithFlags(t *testing.T) {
	var killed string
	err := runOut("", "blue", "vicky", existsChecker, func(name string) error {
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
	err := runOut("blue-vicky", "", "", existsChecker, noopKiller, stopper)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, stoppedContainer, "jack-blue-vicky")
}

func TestRunOutMissingArgs(t *testing.T) {
	err := runOut("", "", "", noopChecker, noopKiller, noopContainerStopper)
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
