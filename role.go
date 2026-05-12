package jack

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileLinker symlinks a file from src to dst.
type FileLinker func(src, dst string) error

// applyAgent ensures the agent's .claude/ directory exists in the agent's
// workspace root (one level above cloned repos). Claude Code's config
// inheritance will merge this with any .claude/ in the repo itself.
func applyAgent(agentName string, ln FileLinker) error {
	agentConfigDir := filepath.Join(env.configDir(), "agents", agentName)
	if _, err := os.Stat(agentConfigDir); err != nil {
		return fmt.Errorf("agent directory not found: agents/%s/", agentName)
	}

	// The agent's .claude/ lives at ~/.jack/{agent}/.claude/ on the host.
	agentClaudeDir := filepath.Join(env.dataDir(), agentName, ".claude")
	if err := os.MkdirAll(agentClaudeDir, 0o750); err != nil {
		return fmt.Errorf("creating agent .claude dir: %w", err)
	}

	return linkDirRecursive(agentConfigDir, agentClaudeDir, ln)
}

// linkDirRecursive symlinks all files from srcDir into dstDir, recursing into
// subdirectories. Existing files at dst are removed before linking.
func linkDirRecursive(srcDir, dstDir string, ln FileLinker) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		if entry.IsDir() {
			if err := os.MkdirAll(dst, 0o750); err != nil {
				return fmt.Errorf("creating directory %s: %w", dst, err)
			}
			if err := linkDirRecursive(src, dst, ln); err != nil {
				return err
			}
			continue
		}

		_ = os.Remove(dst)
		if err := ln(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// linkFile creates a symlink at dst pointing to src. The source path is
// resolved through EvalSymlinks so the link always points to the real file.
func linkFile(src, dst string) error {
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	return os.Symlink(resolved, filepath.Clean(dst))
}
