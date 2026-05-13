package jack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// applyAgent copies the agent's config directory into the agent's workspace
// .claude/ directory. Claude Code's config inheritance will merge this with
// any .claude/ in the repo itself.
func applyAgent(agentName string) error {
	agentConfigDir := filepath.Join(env.configDir(), "agents", agentName)
	if _, err := os.Stat(agentConfigDir); err != nil {
		return fmt.Errorf("agent directory not found: agents/%s/", agentName)
	}

	// The agent's .claude/ lives at ~/.jack/{agent}/.claude/ on the host.
	agentClaudeDir := filepath.Join(env.dataDir(), agentName, ".claude")

	// Remove old agent config and recreate from source.
	_ = os.RemoveAll(agentClaudeDir)
	if err := os.MkdirAll(agentClaudeDir, 0o750); err != nil {
		return fmt.Errorf("creating agent .claude dir: %w", err)
	}

	return copyDirRecursive(agentConfigDir, agentClaudeDir)
}

// copyDirRecursive copies all files from srcDir into dstDir, recursing into
// subdirectories. Symlinks are followed and the target content is copied.
func copyDirRecursive(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())

		// Resolve symlinks to get the real file/dir.
		info, err := os.Stat(src)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if err := os.MkdirAll(dst, 0o750); err != nil {
				return fmt.Errorf("creating directory %s: %w", dst, err)
			}
			if err := copyDirRecursive(src, dst); err != nil {
				return err
			}
			continue
		}

		if err := copyFileContent(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// copyFileContent copies a file's content from src to dst.
func copyFileContent(src, dst string) error {
	in, err := os.Open(filepath.Clean(src))
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(filepath.Clean(dst))
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	return err
}
