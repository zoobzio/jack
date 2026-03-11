package jack

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileCopier copies a file from src to dst.
type FileCopier func(src, dst string) error

// applyTeam provisions governance, orders, project, skill, and agent files into
// the repo's .claude directory.
func applyTeam(teamName, repo, dir string, cp FileCopier) error {
	skills, err := discoverTeamSkills(teamName)
	if err != nil {
		return err
	}

	configDir := env.configDir()
	claudeDir := filepath.Join(dir, ".claude")

	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		return fmt.Errorf("creating .claude dir: %w", err)
	}

	// 1. Governance — copy all files from governance/ to .claude/
	govDir := filepath.Join(configDir, "governance")
	if err := copyDirFiles(govDir, claudeDir, cp); err != nil {
		return fmt.Errorf("copying governance files: %w", err)
	}

	// 2. Orders — copy teams/{teamName}/ORDERS.md to .claude/ORDERS.md
	ordersSrc := filepath.Join(configDir, "teams", teamName, "ORDERS.md")
	ordersDst := filepath.Join(claudeDir, "ORDERS.md")
	if err := cp(ordersSrc, ordersDst); err != nil {
		return fmt.Errorf("copying orders for team %q: %w", teamName, err)
	}

	// 3. Project — copy all files from projects/<repo>/ to .claude/
	projectDir := filepath.Join(configDir, "projects", repo)
	if err := copyDirFiles(projectDir, claudeDir, cp); err != nil {
		return fmt.Errorf("copying project files: %w", err)
	}

	// 4. Skills — resolve symlinks, copy to .claude/commands/
	if len(skills) > 0 {
		commandsDir := filepath.Join(claudeDir, "commands")
		if err := os.MkdirAll(commandsDir, 0o750); err != nil {
			return fmt.Errorf("creating commands dir: %w", err)
		}
		for _, skill := range skills {
			src := filepath.Join(configDir, "teams", teamName, "skills", skill)
			resolved, err := filepath.EvalSymlinks(src)
			if err != nil {
				return fmt.Errorf("resolving skill %q: %w", skill, err)
			}
			dst := filepath.Join(commandsDir, skill)
			if err := copyDirRecursive(resolved, dst, cp); err != nil {
				return fmt.Errorf("copying skill %q: %w", skill, err)
			}
		}
	}

	// 5. Agents — copy from teams/{teamName}/agents/ to .claude/agents/
	agentsSrc := filepath.Join(configDir, "teams", teamName, "agents")
	if entries, err := os.ReadDir(agentsSrc); err == nil && len(entries) > 0 {
		agentsDst := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDst, 0o750); err != nil {
			return fmt.Errorf("creating agents dir: %w", err)
		}
		if err := copyDirFiles(agentsSrc, agentsDst, cp); err != nil {
			return fmt.Errorf("copying agents for team %q: %w", teamName, err)
		}
	}

	return nil
}

// copyDirFiles copies all files (not subdirectories) from srcDir into dstDir.
func copyDirFiles(srcDir, dstDir string, cp FileCopier) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if err := cp(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// copyDirRecursive copies a directory and all its contents into dst.
func copyDirRecursive(srcDir, dstDir string, cp FileCopier) error {
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		return err
	}
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		src := filepath.Join(srcDir, entry.Name())
		dst := filepath.Join(dstDir, entry.Name())
		if entry.IsDir() {
			if err := copyDirRecursive(src, dst, cp); err != nil {
				return err
			}
			continue
		}
		if err := cp(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// validateGovernance checks that governance, team skills, orders, and project
// files exist before any cloning begins.
func validateGovernance(configDir, teamName, repo string) error {
	// Governance directory must exist and be non-empty.
	govDir := filepath.Join(configDir, "governance")
	entries, err := os.ReadDir(govDir)
	if err != nil {
		return fmt.Errorf("governance directory is empty or missing: %w", err)
	}
	hasFiles := false
	for _, e := range entries {
		if !e.IsDir() {
			hasFiles = true
			break
		}
	}
	if !hasFiles {
		return fmt.Errorf("governance directory is empty or missing")
	}

	// Team skills directory must exist.
	teamSkillsDir := filepath.Join(configDir, "teams", teamName, "skills")
	if info, err := os.Stat(teamSkillsDir); err != nil || !info.IsDir() {
		return fmt.Errorf("team skills directory not found: teams/%s/skills/", teamName)
	}

	// Team ORDERS.md must exist.
	ordersPath := filepath.Join(configDir, "teams", teamName, "ORDERS.md")
	if _, err := os.Stat(ordersPath); err != nil {
		return fmt.Errorf("ORDERS.md not found in teams/%s/", teamName)
	}

	// Project directory must exist.
	projectDir := filepath.Join(configDir, "projects", repo)
	if _, err := os.Stat(projectDir); err != nil {
		return fmt.Errorf("project directory not found: projects/%s/", repo)
	}

	// MISSION.md must exist in project directory.
	missionPath := filepath.Join(projectDir, "MISSION.md")
	if _, err := os.Stat(missionPath); err != nil {
		return fmt.Errorf("MISSION.md not found in projects/%s/", repo)
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
