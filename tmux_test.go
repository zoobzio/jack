//go:build testing

package jack

import (
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestSessionName(t *testing.T) {
	jtesting.AssertEqual(t, SessionName("blue", "vicky"), "blue-vicky")
	jtesting.AssertEqual(t, SessionName("red", "flux"), "red-flux")
}

func TestParseSessionName(t *testing.T) {
	teams := []string{"blue", "red"}

	tests := []struct {
		name     string
		input    string
		wantTeam string
		wantRepo string
		wantOK   bool
	}{
		{"valid blue", "blue-vicky", "blue", "vicky", true},
		{"valid red", "red-flux", "red", "flux", true},
		{"repo with hyphens", "blue-my-cool-repo", "blue", "my-cool-repo", true},
		{"unknown team", "green-foo", "", "", false},
		{"no hyphen", "blue", "", "", false},
		{"empty", "", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			team, repo, ok := ParseSessionName(tt.input, teams)
			jtesting.AssertEqual(t, ok, tt.wantOK)
			jtesting.AssertEqual(t, team, tt.wantTeam)
			jtesting.AssertEqual(t, repo, tt.wantRepo)
		})
	}
}

func TestParseSessionNameLongestMatch(t *testing.T) {
	teams := []string{"red", "redshift"}

	team, repo, ok := ParseSessionName("redshift-flux", teams)
	jtesting.AssertEqual(t, ok, true)
	jtesting.AssertEqual(t, team, "redshift")
	jtesting.AssertEqual(t, repo, "flux")
}

func TestParseSessions(t *testing.T) {
	input := "blue-vicky\t1709900000\t1709900060\t/home/user/vicky\t1\t3\n" +
		"personal\t1709800000\t1709800000\t/home/user\t0\t1\n"

	sessions, err := parseSessions(input)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(sessions), 2)
	jtesting.AssertEqual(t, sessions[0].Name, "blue-vicky")
	jtesting.AssertEqual(t, sessions[0].Attached, true)
	jtesting.AssertEqual(t, sessions[0].Windows, 3)
	jtesting.AssertEqual(t, sessions[1].Name, "personal")
	jtesting.AssertEqual(t, sessions[1].Attached, false)
}

func TestParseSessionsEmpty(t *testing.T) {
	sessions, err := parseSessions("")
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(sessions), 0)
}
