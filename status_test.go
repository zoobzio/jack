//go:build testing

package jack

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunStatusNoSessions(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {},
			"red":  {},
		},
	}

	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue"), 0o750)
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "red"), 0o750)

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
}

func TestRunStatusWithSessions(t *testing.T) {
	cfg = Config{
		Profiles: map[string]Profile{
			"blue": {},
			"red":  {},
		},
	}

	configDir := t.TempDir()
	env = Env{ConfigDir: configDir, DataDir: t.TempDir()}
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "blue"), 0o750)
	_ = os.MkdirAll(filepath.Join(configDir, "teams", "red"), 0o750)

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
	// Non-jack session filtered out.
	jtesting.AssertEqual(t, strings.Contains(output, "personal"), false)
}
