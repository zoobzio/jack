package msg

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func init() {
	whoCmd.Flags().Bool("online", false, "show presence, last-seen status, and board membership")
	Cmd.AddCommand(whoCmd)
}

var whoCmd = &cobra.Command{
	Use:   "who",
	Short: "List registered session users",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		online, _ := cmd.Flags().GetBool("online")
		if online {
			token, err := TokenFromEnv()
			if err != nil {
				return err
			}
			client := NewClient(Homeserver, token)
			return runWhoOnline(DataDir, client.GetPresence, client.ResolveAlias, client.Members)
		}
		return runWho(DataDir)
	},
}

// registryData is a minimal representation of the jack registry for reading
// session users without importing the parent package.
type registryData struct {
	Projects []struct {
		Agent string `yaml:"agent"`
		Repo  string `yaml:"repo"`
	} `yaml:"projects"`
}

func loadRegistry(dataDir string) (*registryData, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("data directory not configured")
	}
	regPath := filepath.Clean(filepath.Join(dataDir, "registry.yaml"))
	data, err := os.ReadFile(regPath)
	if os.IsNotExist(err) {
		return &registryData{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading registry: %w", err)
	}
	var reg registryData
	if err := yaml.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing registry: %w", err)
	}
	return &reg, nil
}

func runWho(dataDir string) error {
	reg, err := loadRegistry(dataDir)
	if err != nil {
		return err
	}
	server := ServerName(Homeserver)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range reg.Projects {
		_, _ = fmt.Fprintf(w, "@%s-%s:%s\t%s\t%s\n", p.Agent, p.Repo, server, p.Agent, p.Repo)
	}
	return w.Flush()
}

func runWhoOnline(dataDir string, getPresence PresenceGetter, resolve AliasResolver, members MemberLister) error {
	reg, err := loadRegistry(dataDir)
	if err != nil {
		return err
	}
	server := ServerName(Homeserver)

	// Resolve global board members for the "board" column.
	boardMembers := map[string]bool{}
	alias := boardAlias(GlobalBoardAlias)
	if resp, err := resolve(alias); err == nil {
		if mems, err := members(resp.RoomID); err == nil {
			for _, m := range mems {
				boardMembers[m.UserID] = true
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range reg.Projects {
		userID := fmt.Sprintf("@%s-%s:%s", p.Agent, p.Repo, server)

		status := unknownPlaceholder
		presence, err := getPresence(userID)
		if err == nil {
			status = presence.Presence
			if presence.CurrentlyActive {
				status = "online"
			} else if presence.LastActiveAgo > 0 {
				mins := presence.LastActiveAgo / 60000
				if mins < 1 {
					status += " (<1m ago)"
				} else {
					status += fmt.Sprintf(" (%dm ago)", mins)
				}
			}
		}

		board := "no"
		if boardMembers[userID] {
			board = "yes"
		}

		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\tboard: %s\n", userID, p.Agent, p.Repo, status, board)
	}
	return w.Flush()
}
