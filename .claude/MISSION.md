# Mission: jack

Jack is the operator's console for multi-agent development at zoobz.io.

## Purpose

Running agentic development means running multiple Claude Code instances across teams, repos, and identities. Each team needs its own GitHub account, its own clone of the repo, and the right agent definitions and skills injected into `.claude/`. Sessions need to persist, status needs to be visible, and the operator needs to jack in and out freely.

Jack provides the infrastructure. The operator provides the direction.

## Core Concepts

- **Team** — A named group with a GitHub identity. A team gets its own clone of a repo and its own `.claude/` configuration. Teams are defined in configuration, not hardcoded — jack supports N teams.
- **Sandbox** — A team's clone of a repository. `.claude/` is gitignored in the source repo but jack manages it: cloning the repo, then injecting the correct agent definitions and skills for that team.
- **Session** — A tmux session running Claude Code inside a sandbox. Sessions persist across terminal disconnects. The operator jacks in to interact and detaches when done.
- **Profile** — A GitHub identity (credentials, git config) associated with a team. Commits, PRs, and reviews from a team's sessions come from that team's identity.

## What This Package Contains

- A cobra CLI (`jack`) for managing teams, sandboxes, sessions, and profiles
- tmux session lifecycle: create, list, attach, detach, kill
- Sandbox management: clone repos, inject `.claude/` configuration per team
- Profile management: associate GitHub identities with teams
- Session status: detect which sessions are awaiting operator input via Claude Code notification hooks
- Configuration via file or CLI flags

## What This Package Does NOT Contain

- Workflow orchestration — jack does not tell agents what to do, the operator does
- Agent logic — Claude Code and the agent definitions handle that
- A TUI dashboard (future consideration)
- Remote session management

## Success Criteria

1. Teams are configurable — name, GitHub identity, agent/skill definitions
2. `jack clone` creates a sandbox for a team: clones a repo and injects the team's `.claude/` configuration
3. `jack new` starts a persistent Claude Code session inside a team's sandbox with the correct identity
4. `jack ls` shows all sessions with their team, repo, and input status
5. `jack in` attaches the operator to a running session
6. `jack status` shows sessions awaiting operator input
7. Sessions survive terminal close and can be reattached
8. Multiple teams can work on the same repo independently in their own sandboxes

## Non-Goals

- Managing teams' internal workflow or agent coordination
- Replacing tmux — jack is a thin layer on top
- Supporting non-Claude agent runtimes
- IDE or editor integration
