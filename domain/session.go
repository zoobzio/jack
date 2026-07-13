package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Session is a single tmux session as reported by tmux: its name and the live
// attributes jack uses to describe it (creation and last-activity times, the
// working directory, whether a client is attached, and its window count).
type Session struct {
	Name     string
	Created  time.Time
	Activity time.Time
	Path     string
	Attached bool
	Windows  int // number of tmux windows in the session
}

// NewSession builds a Session from the six tab-separated fields of one
// `tmux list-sessions` line.
func NewSession(fields []string) (Session, error) {
	if len(fields) != 6 {
		return Session{}, fmt.Errorf("unexpected tmux output: %q", strings.Join(fields, "\t"))
	}
	created, err := timestamp(fields[1])
	if err != nil {
		return Session{}, fmt.Errorf("parsing created time: %w", err)
	}
	activity, err := timestamp(fields[2])
	if err != nil {
		return Session{}, fmt.Errorf("parsing activity time: %w", err)
	}
	return Session{
		Name:     fields[0],
		Created:  created,
		Activity: activity,
		Path:     fields[3],
		Attached: fields[4] != "0",
		Windows:  intor(fields[5], 1),
	}, nil
}

// Status returns a short, human-readable state for the session: "attached" if a
// client is attached, "active" if used within the last minute, otherwise
// "idle <duration>".
func (s Session) Status() string {
	if s.Attached {
		return "attached"
	}
	idle := time.Since(s.Activity)
	switch {
	case idle < time.Minute:
		return "active"
	case idle < time.Hour:
		return fmt.Sprintf("idle %dm", int(idle.Minutes()))
	case idle < 24*time.Hour:
		return fmt.Sprintf("idle %dh", int(idle.Hours()))
	default:
		return fmt.Sprintf("idle %dd", int(idle.Hours()/24))
	}
}

// timestamp parses a Unix-seconds string into a time.Time.
func timestamp(s string) (time.Time, error) {
	sec, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(sec, 0), nil
}

// intor parses s as an int, returning fallback if it does not parse.
func intor(s string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return fallback
	}
	return n
}
