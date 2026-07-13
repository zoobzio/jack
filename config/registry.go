package config

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/zoobzio/jack/domain"
)

// RegistryEntry records a repo cloned for an agent.
type RegistryEntry struct {
	ClonedAt time.Time    `yaml:"cloned_at"`
	Agent    domain.Agent `yaml:"agent"`
	Repo     domain.Repo  `yaml:"repo"`
	URL      string       `yaml:"url"`
}

// Registry tracks which repos have been cloned for which agents. It remembers
// the file it was loaded from, so Save writes back to the same place.
type Registry struct {
	path     string
	Projects []RegistryEntry `yaml:"projects"`
}

// NewRegistry loads the registry from path, returning an empty (but saveable)
// registry if the file does not yet exist.
func NewRegistry(path string) (*Registry, error) {
	reg := &Registry{path: path}
	data, err := os.ReadFile(filepath.Clean(path))
	if os.IsNotExist(err) {
		return reg, nil
	}
	if err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(data, reg); err != nil {
		return nil, err
	}
	return reg, nil
}

// Save writes the registry back to the file it was loaded from, creating the
// data directory if needed.
func (r *Registry) Save() error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o600)
}

// Add records a new entry, replacing any existing one for the same agent+repo.
func (r *Registry) Add(agent domain.Agent, repo domain.Repo, url string) {
	r.Remove(agent, repo)
	r.Projects = append(r.Projects, RegistryEntry{
		Agent:    agent,
		Repo:     repo,
		URL:      url,
		ClonedAt: time.Now().UTC().Truncate(time.Second),
	})
}

// Remove deletes the entry for the given agent+repo if it exists.
func (r *Registry) Remove(agent domain.Agent, repo domain.Repo) {
	filtered := r.Projects[:0]
	for _, p := range r.Projects {
		if p.Agent != agent || p.Repo != repo {
			filtered = append(filtered, p)
		}
	}
	r.Projects = filtered
}

// Find returns the entry for the given agent+repo, or nil if not found.
func (r *Registry) Find(agent domain.Agent, repo domain.Repo) *RegistryEntry {
	for i := range r.Projects {
		if r.Projects[i].Agent == agent && r.Projects[i].Repo == repo {
			return &r.Projects[i]
		}
	}
	return nil
}

// ForAgent returns all entries for the given agent, sorted by repo name.
func (r *Registry) ForAgent(agent domain.Agent) []RegistryEntry {
	var entries []RegistryEntry
	for _, p := range r.Projects {
		if p.Agent == agent {
			entries = append(entries, p)
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Repo < entries[j].Repo
	})
	return entries
}

// Agents returns the sorted, unique agent names in the registry.
func (r *Registry) Agents() []domain.Agent {
	seen := make(map[domain.Agent]bool)
	for _, p := range r.Projects {
		seen[p.Agent] = true
	}
	agents := make([]domain.Agent, 0, len(seen))
	for a := range seen {
		agents = append(agents, a)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i] < agents[j] })
	return agents
}

// ReposForAgent returns the sorted repo names cloned for the given agent.
func (r *Registry) ReposForAgent(agent domain.Agent) []domain.Repo {
	var repos []domain.Repo
	for _, p := range r.Projects {
		if p.Agent == agent {
			repos = append(repos, p.Repo)
		}
	}
	sort.Slice(repos, func(i, j int) bool { return repos[i] < repos[j] })
	return repos
}
