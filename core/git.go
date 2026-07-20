package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Git is the boundary to the host git CLI. It performs I/O only, against clones
// on the host filesystem: it clones repos and sets local config.
type Git interface {
	// Clone clones url into dir, streaming progress to the terminal.
	Clone(ctx context.Context, url, dir string) error
	// Config sets a local git config value in the clone at dir.
	Config(ctx context.Context, dir, key, value string) error
	// GlobalIdentity reads the host's global git user.name and user.email. An
	// unset value comes back empty rather than as an error, so callers can seed
	// config from whatever identity exists.
	GlobalIdentity(ctx context.Context) (name, email string, err error)
}

// git is the real Git implementation, backed by the local git CLI. It holds no
// state — every method shells out to git.
type git struct{}

// NewGit returns a Git backed by the local git CLI.
func NewGit() Git {
	return &git{}
}

// Clone runs `git clone`, streaming its output to the terminal so the user sees
// progress.
func (g *git) Clone(ctx context.Context, url, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "clone", url, dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	return nil
}

// Config runs `git -C dir config key value`. Errors carry git's stderr.
func (g *git) Config(ctx context.Context, dir, key, value string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "config", key, value)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git config %s: %w: %s", key, err, stderr.String())
	}

	return nil
}

// GlobalIdentity reads `git config --global` for user.name and user.email. An
// unset key makes git exit non-zero; that is reported as an empty value, not an
// error, so a host with no global identity still yields a usable (empty) result.
// A missing git binary surfaces as an error from the first lookup.
func (g *git) GlobalIdentity(ctx context.Context) (name, email string, err error) {
	name, err = g.globalConfig(ctx, "user.name")
	if err != nil {
		return "", "", err
	}
	email, err = g.globalConfig(ctx, "user.email")
	if err != nil {
		return "", "", err
	}
	return name, email, nil
}

// globalConfig returns the value of a single `git config --global` key, or an
// empty string when the key is unset. It distinguishes "key not set" (git exits
// with an error but nothing on stderr) from a real failure to run git.
func (g *git) globalConfig(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "config", "--global", key) // #nosec G204 -- key is a static config name
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// An unset key exits non-zero with no diagnostic; treat that as empty.
		if stderr.Len() == 0 {
			return "", nil
		}
		return "", fmt.Errorf("git config --global %s: %w: %s", key, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Compile-time assertion that *git satisfies the Git interface.
var _ Git = (*git)(nil)
