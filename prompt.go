package jack

import (
	"fmt"

	"github.com/charmbracelet/huh"
)

// AgentSelector prompts the user to select an agent.
type AgentSelector func(agents []string) (string, error)

// ProjectSelector prompts the user to select a project for an agent.
type ProjectSelector func(agent string, repos []string) (string, error)

// selectRunner abstracts huh form execution for testing.
type selectRunner func(opts []string, title string) (string, error)

// defaultSelectRunner runs a huh select form.
func defaultSelectRunner(opts []string, title string) (string, error) {
	var selected string
	err := huh.NewSelect[string]().
		Title(title).
		Options(huh.NewOptions(opts...)...).
		Value(&selected).
		Run()
	return selected, err
}

var runSelect selectRunner = defaultSelectRunner

func selectAgent(agents []string) (string, error) {
	agent, err := runSelect(agents, "Select an agent")
	if err != nil {
		return "", fmt.Errorf("selecting agent: %w", err)
	}
	return agent, nil
}

func selectProject(agent string, repos []string) (string, error) {
	repo, err := runSelect(repos, fmt.Sprintf("Select a project for %s", agent))
	if err != nil {
		return "", fmt.Errorf("selecting project: %w", err)
	}
	return repo, nil
}
