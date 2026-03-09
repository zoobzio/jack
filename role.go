package jack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileCopier copies a file from src to dst.
type FileCopier func(src, dst string) error

// applyRole copies the role's skills and team's agents into the repo's .claude directory.
func applyRole(roleName, teamName, dir string, cp FileCopier) error {
	role, ok := cfg.Roles[roleName]
	if !ok {
		return fmt.Errorf("unknown role %q", roleName)
	}

	team, ok := cfg.Teams[teamName]
	if !ok {
		return fmt.Errorf("unknown team %q", teamName)
	}

	configDir := env.configDir()

	// Copy skills to .claude/commands/
	if len(role.Skills) > 0 {
		commandsDir := filepath.Join(dir, ".claude", "commands")
		if err := os.MkdirAll(commandsDir, 0o750); err != nil {
			return fmt.Errorf("creating commands dir: %w", err)
		}
		for _, skill := range role.Skills {
			src := filepath.Join(configDir, "skills", skill+".md")
			dst := filepath.Join(commandsDir, skill+".md")
			if err := cp(src, dst); err != nil {
				return fmt.Errorf("copying skill %q: %w", skill, err)
			}
		}
	}

	// Copy agents to .claude/agents/
	if len(team.Agents) > 0 {
		agentsDir := filepath.Join(dir, ".claude", "agents")
		if err := os.MkdirAll(agentsDir, 0o750); err != nil {
			return fmt.Errorf("creating agents dir: %w", err)
		}
		for _, agent := range team.Agents {
			src := filepath.Join(configDir, "agents", agent+".md")
			dst := filepath.Join(agentsDir, agent+".md")
			if err := cp(src, dst); err != nil {
				return fmt.Errorf("copying agent %q: %w", agent, err)
			}
		}
	}

	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
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

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
