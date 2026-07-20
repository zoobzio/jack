package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
)

// Git is the boundary to the host git CLI. It performs I/O only, against clones
// on the host filesystem: it clones repos and sets local config.
type Git interface {
	// Clone clones url into dir, streaming progress to the terminal.
	Clone(ctx context.Context, url, dir string) error
	// Config sets a local git config value in the clone at dir.
	Config(ctx context.Context, dir, key, value string) error
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

// Compile-time assertion that *git satisfies the Git interface.
var _ Git = (*git)(nil)
