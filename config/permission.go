package config

import "fmt"

// Permission is the Claude Code permission mode an agent's session launches in.
// It maps to the `claude` CLI flags jack appends to the launch command.
type Permission string

const (
	// PermissionDefault is Claude Code's manual mode: it prompts before edits and
	// commands. It is the zero value and adds no launch flag.
	PermissionDefault Permission = "default"
	// PermissionAcceptEdits auto-accepts file edits but still gates risky commands.
	PermissionAcceptEdits Permission = "acceptEdits"
	// PermissionBypass skips all permission checks — appropriate for jack's
	// isolated per-agent containers. It maps to --dangerously-skip-permissions.
	PermissionBypass Permission = "bypassPermissions"
)

// Validate reports whether the permission is one jack understands. The empty
// string is allowed and means the default (manual) mode.
func (p Permission) Validate() error {
	switch p {
	case "", PermissionDefault, PermissionAcceptEdits, PermissionBypass:
		return nil
	default:
		return fmt.Errorf("invalid permission %q (want %q, %q, or %q)",
			p, PermissionDefault, PermissionAcceptEdits, PermissionBypass)
	}
}

// Flags returns the `claude` CLI arguments for this permission mode, or an empty
// string for the default (manual) mode, which needs no flag.
func (p Permission) Flags() string {
	switch p {
	case PermissionAcceptEdits:
		return "--permission-mode acceptEdits"
	case PermissionBypass:
		return "--dangerously-skip-permissions"
	default:
		return ""
	}
}
