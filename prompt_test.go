//go:build testing

package jack

import (
	"fmt"
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestSelectAgentSuccess(t *testing.T) {
	orig := runSelect
	defer func() { runSelect = orig }()

	runSelect = func(opts []string, title string) (string, error) {
		jtesting.AssertEqual(t, len(opts), 2)
		jtesting.AssertEqual(t, title, "Select an agent")
		return opts[1], nil
	}

	agent, err := selectAgent([]string{"blue", "red"})
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, agent, "red")
}

func TestSelectAgentError(t *testing.T) {
	orig := runSelect
	defer func() { runSelect = orig }()

	runSelect = func(_ []string, _ string) (string, error) {
		return "", fmt.Errorf("user cancelled")
	}

	_, err := selectAgent([]string{"blue"})
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "selecting agent"), true)
}

func TestSelectProjectSuccess(t *testing.T) {
	orig := runSelect
	defer func() { runSelect = orig }()

	runSelect = func(opts []string, title string) (string, error) {
		jtesting.AssertEqual(t, strings.Contains(title, "blue"), true)
		return opts[0], nil
	}

	repo, err := selectProject("blue", []string{"vicky", "flux"})
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, repo, "vicky")
}

func TestSelectProjectError(t *testing.T) {
	orig := runSelect
	defer func() { runSelect = orig }()

	runSelect = func(_ []string, _ string) (string, error) {
		return "", fmt.Errorf("interrupted")
	}

	_, err := selectProject("blue", []string{"vicky"})
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "selecting project"), true)
}
