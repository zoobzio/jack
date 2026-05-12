//go:build testing

package jack

import (
	"strings"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestSessionName(t *testing.T) {
	jtesting.AssertEqual(t, SessionName("blue", "vicky", ""), "blue-vicky")
	jtesting.AssertEqual(t, SessionName("red", "flux", ""), "red-flux")
}

func TestSessionNameWithWorktree(t *testing.T) {
	name := SessionName("blue", "vicky", "feature-auth")
	hash := WorktreeHash("feature-auth")
	jtesting.AssertEqual(t, name, "blue-vicky-"+hash)
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

func TestParseSessionsWrongFields(t *testing.T) {
	// Only 3 fields instead of 6.
	_, err := parseSessions("name\tfield2\tfield3\n")
	jtesting.AssertError(t, err)
}

func TestParseSessionsEmptyLines(t *testing.T) {
	// Input with blank line between valid lines — should skip the blank.
	input := "blue-vicky\t1709900000\t1709900060\t/home/user/vicky\t1\t3\n\npersonal\t1709800000\t1709800000\t/home/user\t0\t1"
	sessions, err := parseSessions(input)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, len(sessions), 2)
	jtesting.AssertEqual(t, sessions[0].Name, "blue-vicky")
	jtesting.AssertEqual(t, sessions[1].Name, "personal")
}

func TestParseTmuxSessionActivityError(t *testing.T) {
	// Valid created timestamp but invalid activity timestamp.
	parts := []string{"name", "1709900000", "notanumber", "/path", "0", "1"}
	_, err := parseTmuxSession(parts)
	jtesting.AssertError(t, err)
}

func TestParseTmuxSessionCreatedError(t *testing.T) {
	// Invalid created timestamp.
	parts := []string{"name", "notanumber", "1709900000", "/path", "0", "1"}
	_, err := parseTmuxSession(parts)
	jtesting.AssertError(t, err)
	jtesting.AssertEqual(t, strings.Contains(err.Error(), "parsing created time"), true)
}

func TestParseSessionsBadTimestamp(t *testing.T) {
	// Valid format (6 fields) but parseTmuxSession fails — tests the error branch in parseSessions.
	input := "session\tnotanumber\t1709900000\t/path\t0\t1\n"
	_, err := parseSessions(input)
	jtesting.AssertError(t, err)
}

func TestParseUnixTimestampError(t *testing.T) {
	_, err := parseUnixTimestamp("not-a-number")
	jtesting.AssertError(t, err)
}

func TestParseIntOrFallback(t *testing.T) {
	jtesting.AssertEqual(t, parseIntOr("notanumber", 42), 42)
	jtesting.AssertEqual(t, parseIntOr("7", 42), 7)
}
