//go:build testing

package jack

import (
	"bytes"
	"strings"
	"testing"
	"time"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunStatusNoSessions(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"rockhopper": {},
			"mother":     {},
		},
		Teams: map[string]Team{
			"blue": {Profile: "rockhopper"},
			"red":  {Profile: "mother"},
		},
	}

	var buf bytes.Buffer
	err := runStatus(&buf, func() ([]TmuxSession, error) {
		return nil, nil
	})
	jtesting.AssertNoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Header + 2 teams.
	jtesting.AssertEqual(t, len(lines), 3)
	jtesting.AssertEqual(t, strings.Contains(output, "blue"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "red"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "rockhopper"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "mother"), true)
}

func TestRunStatusWithSessions(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"rockhopper": {},
			"mother":     {},
		},
		Teams: map[string]Team{
			"blue": {Profile: "rockhopper"},
			"red":  {Profile: "mother"},
		},
	}

	var buf bytes.Buffer
	err := runStatus(&buf, func() ([]TmuxSession, error) {
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
	})
	jtesting.AssertNoError(t, err)

	output := buf.String()
	// Both blue sessions appear with details.
	jtesting.AssertEqual(t, strings.Contains(output, "blue-vicky"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "blue-flux"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "/home/user/vicky"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "attached"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "active"), true)
	// Red has no sessions, still listed.
	jtesting.AssertEqual(t, strings.Contains(output, "red"), true)
	jtesting.AssertEqual(t, strings.Contains(output, "mother"), true)
	// Non-jack session filtered out.
	jtesting.AssertEqual(t, strings.Contains(output, "personal"), false)
}
