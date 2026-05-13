package jack

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const jackImage = "jack"
const containerHome = "/root"

const baseDockerfile = `FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN npm install -g @anthropic-ai/claude-code
RUN mkdir -p /root/.jack/bin /root/workspace
ENV PATH="/root/.jack/bin:${PATH}"
WORKDIR /root
`

// Mount describes a Docker bind mount.
type Mount struct {
	Source   string
	Target  string
	ReadOnly bool
}

// ImageBuilder builds the jack base Docker image.
type ImageBuilder func(ctx context.Context) error

// ContainerRunner starts an idle container with the given mounts and env.
type ContainerRunner func(name string, mounts []Mount, volumes []Volume, env map[string]string) error

// ContainerStopper stops and removes a container.
type ContainerStopper func(name string) error

// ContainerExecer runs a command inside a running container.
type ContainerExecer func(name string, cmd []string) error

// ContainerChecker reports whether a container is running and/or exists.
type ContainerChecker func(name string) (running bool, exists bool)

// Volume describes a named Docker volume mount.
type Volume struct {
	Name   string
	Target string
}

// ContainerName builds the canonical Docker container name for an agent and repo.
func ContainerName(agent, repo string) string {
	return "jack-" + agent + "-" + repo
}

// WorktreeContainerPath returns the container path for a worktree.
func WorktreeContainerPath(repo, branch string) string {
	return containerHome + "/workspace/" + WorktreeDir(repo, branch)
}

// ToolsVolume returns the named volume for persisting installed tools.
func ToolsVolume(agent, repo string) Volume {
	return Volume{
		Name:   "jack-" + agent + "-" + repo + "-tools",
		Target: containerHome + "/.jack/bin",
	}
}

// SessionMounts returns the standard bind mounts for a session container.
// The agent's .claude/ is mounted one level above the repo so that Claude
// Code's config inheritance merges agent config with the repo's own .claude/.
func SessionMounts(profile Profile, agent, repo, repoDir string) []Mount {
	home, _ := os.UserHomeDir()
	agentClaudeDir := filepath.Join(env.dataDir(), agent, ".claude")
	mounts := []Mount{
		{Source: filepath.Join(home, ".claude"), Target: containerHome + "/.claude", ReadOnly: false},
		{Source: filepath.Join(home, ".claude.json"), Target: containerHome + "/.claude.json", ReadOnly: false},
		{Source: agentClaudeDir, Target: containerHome + "/workspace/.claude", ReadOnly: true},
		{Source: repoDir, Target: containerHome + "/workspace/" + repo, ReadOnly: false},
	}

	// Mount agent certificate and CA root for mTLS authentication.
	if hasCert(agent) {
		mounts = append(mounts,
			Mount{Source: certPath(agent), Target: containerHome + "/.jack/cert.pem", ReadOnly: true},
			Mount{Source: keyPath(agent), Target: containerHome + "/.jack/key.pem", ReadOnly: true},
		)
	}
	if cfg.CA.Root != "" {
		rootPath := expandHome(cfg.CA.Root)
		if _, err := os.Stat(rootPath); err == nil {
			mounts = append(mounts, Mount{Source: rootPath, Target: containerHome + "/.jack/ca.pem", ReadOnly: true})
		}
	}

	// Mount supporting repos.
	for _, repoURL := range profile.Repos {
		name := repoName(repoURL)
		if name == "" {
			continue
		}
		supportDir := filepath.Join(env.dataDir(), agent, name)
		if _, err := os.Stat(supportDir); err == nil {
			mounts = append(mounts, Mount{Source: supportDir, Target: "/repos/" + name, ReadOnly: false})
		}
	}

	return mounts
}

// SessionEnv returns the environment variables for a session container.
func SessionEnv(profile Profile, agent string) map[string]string {
	e := make(map[string]string)
	if agent != "" {
		e["JACK_AGENT"] = agent
	}
	if profile.Git.Name != "" {
		e["GIT_AUTHOR_NAME"] = profile.Git.Name
		e["GIT_COMMITTER_NAME"] = profile.Git.Name
	}
	if profile.Git.Email != "" {
		e["GIT_AUTHOR_EMAIL"] = profile.Git.Email
		e["GIT_COMMITTER_EMAIL"] = profile.Git.Email
	}
	return e
}

// DockerBuild builds the jack base image.
func DockerBuild(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "jack-docker-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	dockerfilePath := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(baseDockerfile), 0o600); err != nil {
		return fmt.Errorf("writing Dockerfile: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", jackImage, dir) // #nosec G204 -- args are static
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}
	return nil
}

// DockerRun starts an idle container with the given name, mounts, volumes, and env.
func DockerRun(name string, mounts []Mount, volumes []Volume, envVars map[string]string) error {
	args := make([]string, 0, 6+2*len(mounts)+2*len(volumes)+2*len(envVars)+3)
	args = append(args, "run", "-d", "--name", name)
	for _, m := range mounts {
		vol := m.Source + ":" + m.Target
		if m.ReadOnly {
			vol += ":ro"
		}
		args = append(args, "-v", vol)
	}
	for _, v := range volumes {
		args = append(args, "-v", v.Name+":"+v.Target)
	}
	for k, v := range envVars {
		args = append(args, "-e", k+"="+v)
	}
	args = append(args, jackImage, "sleep", "infinity")

	cmd := exec.CommandContext(context.Background(), "docker", args...) // #nosec G204 -- args from internal config
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run: %w: %s", err, stderr.String())
	}
	return nil
}

// DockerExec runs a command inside a running container, streaming output.
func DockerExec(name string, cmdArgs []string) error {
	args := make([]string, 0, 2+len(cmdArgs))
	args = append(args, "exec", name)
	args = append(args, cmdArgs...)
	cmd := exec.CommandContext(context.Background(), "docker", args...) // #nosec G204 -- args from internal config
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker exec: %w", err)
	}
	return nil
}

// DockerStop stops and removes a container.
func DockerStop(name string) error {
	cmd := exec.CommandContext(context.Background(), "docker", "rm", "-f", name)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rm: %w: %s", err, stderr.String())
	}
	return nil
}

// DockerCheck reports whether a container is running and whether it exists.
func DockerCheck(name string) (running bool, exists bool) {
	cmd := exec.CommandContext(context.Background(), "docker", "inspect", "--format", "{{.State.Running}}", name)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false, false
	}
	state := strings.TrimSpace(stdout.String())
	return state == "true", true
}

// DockerExecCmd returns the tmux command string that execs into a container.
func DockerExecCmd(container, workdir string, args ...string) string {
	parts := []string{"docker", "exec", "-it", "-w", workdir, container}
	parts = append(parts, args...)
	return strings.Join(parts, " ")
}
