package domain

import "fmt"

// ContainerHome is the root path inside a jack container — the base that the
// container's workspace and tool paths hang off of. It is jack's contract with
// the base image (its WORKDIR and mkdir'd layout), so both the container Spec
// builder and the path methods below derive from this single value.
const ContainerHome = "/root"

// Identity is the canonical name set for a single unit of work, derived from an
// agent and a repo. The raw inputs are retained (unexported) so an identity can
// be matched or reconstructed; the exported fields hold the names jack uses to
// address Docker and tmux.
type Identity struct {
	agent Agent
	repo  Repo

	Container string // Docker container name
	Session   string // tmux session name
}

// NewIdentity validates the agent and repo and builds their derived names. It
// returns an error only if the agent or repo is invalid (see Agent.Validate and
// Repo.Validate).
func NewIdentity(agent Agent, repo Repo) (*Identity, error) {
	if err := agent.Validate(); err != nil {
		return nil, err
	}
	if err := repo.Validate(); err != nil {
		return nil, err
	}

	return &Identity{
		agent:     agent,
		repo:      repo,
		Container: fmt.Sprintf("jack-%s-%s", agent, repo),
		Session:   fmt.Sprintf("%s-%s", agent, repo),
	}, nil
}

// Agent returns the agent this identity was built from. It exists so callers in
// other packages (e.g. the Spec builder) can read the agent without the raw
// field being exported.
func (i *Identity) Agent() Agent { return i.agent }

// Repo returns the repo this identity was built from, the counterpart to Agent.
func (i *Identity) Repo() Repo { return i.repo }

// RepoPath returns the repo's path inside the container — where the repo is
// mounted and where a session's shell starts.
func (i *Identity) RepoPath() string {
	return ContainerHome + "/workspace/" + string(i.repo)
}

// ToolsVolume returns the name of the persistent Docker volume that holds the
// session's installed tools. It is derived from the container name so the Spec
// builder and teardown agree on it.
func (i *Identity) ToolsVolume() string {
	return i.Container + "-tools"
}
