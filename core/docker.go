package core

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const image = "jack"

const dockerfile = `FROM node:22-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN curl -fsSL https://dl.smallstep.com/install-step-cli.sh | bash
RUN npm install -g @anthropic-ai/claude-code
RUN mkdir -p /root/.jack/bin /root/.jack/certs /root/workspace
ENV PATH="/root/.jack/bin:${PATH}"
WORKDIR /root
`

// Docker is the boundary to the docker CLI. It performs I/O only: it executes
// the given actions and reports container state, but does not decide what to
// mount or which env vars to set — that policy lives in the pure builders that
// produce a Spec.
type Docker interface {
	// Build builds the jack base image.
	Build(ctx context.Context) error
	// Run starts a detached container described by spec.
	Run(ctx context.Context, spec Spec) error
	// Exec runs a command inside a running container.
	Exec(ctx context.Context, name string, cmd []string) error
	// Stop stops and removes a container.
	Stop(ctx context.Context, name string) error
	// Running reports whether the named container is currently running. A
	// stopped container returns (false, nil); a container that does not exist
	// (or a docker failure) returns a non-nil error.
	Running(ctx context.Context, name string) (bool, error)
}

// docker is the real Docker implementation, backed by the local docker CLI.
type docker struct {
	image      string
	dockerfile string
}

// NewDocker returns a Docker backed by the local docker CLI, configured to
// build and run jack's base image. The base image provides Node, git/curl, the
// step CLI, and Claude Code, plus the /root layout the Spec builders target —
// its paths (WORKDIR, the mkdir'd dirs) are the contract the mount targets must
// match.
func NewDocker() *docker {
	return &docker{
		image:      image,
		dockerfile: dockerfile,
	}
}

// Build writes the base Dockerfile to a temp dir and runs `docker build`,
// streaming output to the terminal.
func (d *docker) Build(ctx context.Context) error {
	dir, err := os.MkdirTemp("", "jack-docker-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	dockerfilePath := filepath.Join(dir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(d.dockerfile), 0o600); err != nil {
		return fmt.Errorf("writing Dockerfile: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", d.image, dir) // #nosec G204 -- args are static
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	return nil
}

// Run formats spec into `docker run -d` arguments (mounts and volumes as -v,
// env as -e) and starts the container detached, kept alive with `sleep
// infinity` so it can be exec'd into. Errors carry docker's stderr.
func (d *docker) Run(ctx context.Context, spec Spec) error {
	args := make([]string, 0, 6+2*len(spec.Mounts)+2*len(spec.Volumes)+2*len(spec.Env)+3)
	args = append(args, "run", "-d", "--name", spec.Name)

	for _, m := range spec.Mounts {
		vol := m.Source + ":" + m.Target
		if m.ReadOnly {
			vol += ":ro"
		}
		args = append(args, "-v", vol)
	}

	for _, v := range spec.Volumes {
		args = append(args, "-v", v.Name+":"+v.Target)
	}

	for k, v := range spec.Env {
		args = append(args, "-e", k+"="+v)
	}

	args = append(args, d.image, "sleep", "infinity")

	cmd := exec.CommandContext(ctx, "docker", args...) // #nosec G204 -- args from internal config
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker run: %w: %s", err, stderr.String())
	}

	return nil
}

// Exec runs cmd inside the named container via `docker exec`, streaming its
// stdout and stderr to the terminal.
func (d *docker) Exec(ctx context.Context, name string, cmd []string) error {
	args := make([]string, 0, 2+len(cmd))
	args = append(args, "exec", name)
	args = append(args, cmd...)

	exe := exec.CommandContext(ctx, "docker", args...) // #nosec G204 -- args from internal config
	exe.Stdout = os.Stdout
	exe.Stderr = os.Stderr

	if err := exe.Run(); err != nil {
		return fmt.Errorf("docker exec: %w", err)
	}

	return nil
}

// Stop force-removes the named container with `docker rm -f`, which stops it if
// running. Errors carry docker's stderr.
func (d *docker) Stop(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "docker", "rm", "-f", name)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker rm: %w: %s", err, stderr.String())
	}

	return nil
}

// Running inspects the container's .State.Running via `docker inspect`. A
// non-existent container makes inspect fail, which surfaces as an error (see
// the interface contract).
func (d *docker) Running(ctx context.Context, name string) (bool, error) {
	cmd := exec.CommandContext(ctx, "docker", "inspect", "--format", "{{.State.Running}}", name)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return false, err
	}

	state := strings.TrimSpace(stdout.String())

	return state == "true", nil
}

// Compile-time assertion that *docker satisfies the Docker interface.
var _ Docker = (*docker)(nil)
