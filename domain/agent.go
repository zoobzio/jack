package domain

import (
	"errors"
	"strings"
)

// Agent is an agent's name — the identifier that ties a profile to its
// containers and sessions. It is a distinct type so that its validation rule
// travels with it rather than living in a generic helper.
type Agent string

// Validate reports whether the agent name is usable. An agent must be non-empty
// and must not contain '-', because '-' is the delimiter that joins agent,
// repo, and branch hash into container and session names (and splits them back
// out).
func (a Agent) Validate() error {
	if a == "" {
		return errors.New("an agent must not be an empty string")
	}
	if strings.Contains(string(a), "-") {
		return errors.New("an agent must not contain '-' characters")
	}
	return nil
}
