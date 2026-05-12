package jack

import (
	"crypto/sha256"
	"fmt"
)

// WorktreeHash returns a short deterministic hash of a branch name,
// used for session names and worktree directory names.
func WorktreeHash(branch string) string {
	h := sha256.Sum256([]byte(branch))
	return fmt.Sprintf("%x", h[:4])
}

// WorktreeDir returns the directory name for a worktree inside the container.
func WorktreeDir(repo, branch string) string {
	return repo + "-" + WorktreeHash(branch)
}
