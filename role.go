package jack

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileLinker symlinks a file from src to dst.
type FileLinker func(src, dst string) error

// applyTeam provisions governance, orders, project, skill, and agent files into
// the repo's .claude directory. All files are symlinked so the jack config
// directory remains the single source of truth.
func applyTeam(teamName, repo, dir string, ln FileLinker) error {
	skills, err := discoverTeamSkills(teamName)
	if err != nil {
		return err
	}

	configDir := env.configDir()
	claudeDir := filepath.Join(dir, ".claude")

	if err := os.MkdirAll(claudeDir, 0o750); err != nil {
		return fmt.Errorf("creating .claude dir: %w", err)
	}

	// 1. Governance — symlink all files from governance/ to .claude/
	govDir := filepath.Join(configDir, "governance")
	if err := linkDirFiles(govDir, claudeDir, ln); err != nil {
		return fmt.Errorf("linking governance files: %w", err)
	}

	// 2. Team files — symlink all files from teams/{teamName}/ to .claude/
	teamDir := filepath.Join(configDir, "teams", teamName)
	if err := linkDirFiles(teamDir, claudeDir, ln); err != nil {
		return fmt.Errorf("linking team files for %q: %w", teamName, err)
	}

	// 3. Project — symlink all files from projects/<repo>/ to .claude/
	projectDir := filepath.Join(configDir, "projects", repo)
	if err := linkDirFiles(projectDir, claudeDir, ln); err != nil {
		return fmt.Errorf("linking project files: %w", err)
	}

	// 4. Skills — symlink into .claude/commands/ so the config dir stays
	//    the source of truth and skill updates propagate immediately.
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
			if err := os.Symlink(resolved, dst); err != nil {
				return fmt.Errorf("linking skill %q: %w", skill, err)
			}
		}
	}

	// 5. Agents — symlink from teams/{teamName}/agents/ to .claude/agents/
	agentsSrc := filepath.Join(configDir, "teams", teamName, "agents")
	if entries, err := os.ReadDir(agentsSrc); err == nil && len(entries) > 0 {
		agentsDst := filepath.Join(claudeDir, "agents")
		if err := os.MkdirAll(agentsDst, 0o750); err != nil {
			return fmt.Errorf("creating agents dir: %w", err)
		}
		if err := linkDirFiles(agentsSrc, agentsDst, ln); err != nil {
			return fmt.Errorf("linking agents for team %q: %w", teamName, err)
		}
	}

	return nil
}

// linkDirFiles symlinks all files (not subdirectories) from srcDir into dstDir.
// If a file already exists at dst it is removed before linking, allowing
// higher-priority sources (project > team > governance) to override.
func linkDirFiles(srcDir, dstDir string, ln FileLinker) error {
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
		_ = os.Remove(dst)
		if err := ln(src, dst); err != nil {
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

// linkFile creates a symlink at dst pointing to src. The source path is
// resolved through EvalSymlinks so the link always points to the real file.
func linkFile(src, dst string) error {
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}
	return os.Symlink(resolved, filepath.Clean(dst))
}
