package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/zoobzio/jack/domain"
)

// Tmux is the boundary to the tmux CLI. It performs I/O only: it creates,
// inspects, attaches to, and kills sessions, but holds no policy about how
// sessions are named — that comes from the identity.
type Tmux interface {
	// Has reports whether a session with the given name exists.
	Has(ctx context.Context, name string) (bool, error)
	// Create starts a detached session running the given shell command.
	Create(ctx context.Context, name, cmd string) error
	// Attach attaches to an existing session, taking over the terminal.
	Attach(ctx context.Context, name string) error
	// Kill terminates a session.
	Kill(ctx context.Context, name string) error
	// List returns all tmux sessions, or an empty slice if no server is running.
	List(ctx context.Context) ([]domain.Session, error)
}

// tmux is the real Tmux implementation, backed by the local tmux CLI. It holds
// no state — every method shells out to tmux.
type tmux struct{}

// NewTmux returns a Tmux backed by the local tmux CLI.
func NewTmux() Tmux {
	return &tmux{}
}

// Has runs `tmux has-session`. An absent session (or no server) is reported as
// (false, nil), matching List's treatment of "no server" as normal; only a
// genuine failure to run the check returns a non-nil error.
func (t *tmux) Has(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "tmux", "has-session", "-t", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if strings.Contains(msg, "can't find session") ||
			strings.Contains(msg, "no server running") ||
			strings.Contains(msg, "error connecting") {
			return false, nil
		}
		return false, fmt.Errorf("tmux has-session: %w: %s", err, msg)
	}

	return true, nil
}

// Create runs `tmux new-session -d`, starting a detached session that runs cmd
// under `sh -c`. Errors carry tmux's stderr.
func (t *tmux) Create(ctx context.Context, name, cmd string) error {
	exe := exec.CommandContext(ctx, "tmux", "new-session", "-d", "-s", name, "sh", "-c", cmd)
	var stderr bytes.Buffer
	exe.Stderr = &stderr

	if err := exe.Run(); err != nil {
		return fmt.Errorf("tmux new-session: %w: %s", err, stderr.String())
	}

	return nil
}

// Attach runs `tmux attach-session`, wiring the current terminal to the session
// so the caller's process is taken over until the user detaches.
func (t *tmux) Attach(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "tmux", "attach-session", "-t", name)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Kill runs `tmux kill-session`. Errors carry tmux's stderr.
func (t *tmux) Kill(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "tmux", "kill-session", "-t", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tmux kill-session: %w: %s", err, stderr.String())
	}

	return nil
}

// List runs `tmux list-sessions` with a tab-separated format and parses the
// result. A stopped or absent tmux server is not an error — it yields an empty
// slice.
func (t *tmux) List(ctx context.Context) ([]domain.Session, error) {
	format := "#{session_name}\t#{session_created}\t#{session_activity}\t#{session_path}\t#{session_attached}\t#{session_windows}"
	cmd := exec.CommandContext(ctx, "tmux", "list-sessions", "-F", format)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := stderr.String()
		if strings.Contains(msg, "no server running") ||
			strings.Contains(msg, "no sessions") ||
			strings.Contains(msg, "error connecting") {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-sessions: %w: %s", err, msg)
	}

	return t.parse(stdout.String())
}

// parse splits the tab-separated output of `tmux list-sessions` into one Session
// per non-empty line.
func (t *tmux) parse(output string) ([]domain.Session, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	Sessions := make([]domain.Session, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		c, err := domain.NewSession(strings.SplitN(line, "\t", 6))
		if err != nil {
			return nil, err
		}
		Sessions = append(Sessions, c)
	}
	return Sessions, nil
}

// Compile-time assertion that *tmux satisfies the Tmux interface.
var _ Tmux = (*tmux)(nil)
