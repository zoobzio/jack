package msg

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var boardCmd = &cobra.Command{
	Use:   "board",
	Short: "Construct board messaging",
	Long:  "Post to, read from, and watch the agent or global construct board room.",
}

var boardPostCmd = &cobra.Command{
	Use:   "post <message...>",
	Short: "Post a message to the board",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		message := strings.Join(args, " ")
		name, topic, aliasName, err := boardTarget(cmd)
		if err != nil {
			return err
		}
		if err := runBoardPost(name, topic, aliasName, message, client.ResolveAlias, client.Send, client.CreateRoomWithAlias); err != nil {
			return err
		}
		return postCheck(cmd)
	},
}

var boardReadCmd = &cobra.Command{
	Use:   "read",
	Short: "Read messages from the board",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		jsonFlag, _ := cmd.Flags().GetBool("json")
		since, _ := cmd.Flags().GetString("since")
		from, _ := cmd.Flags().GetString("from")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		name, topic, aliasName, err := boardTarget(cmd)
		if err != nil {
			return err
		}
		if since != "" {
			roomID, err := ensureBoardRoom(name, topic, aliasName, client.ResolveAlias, client.CreateRoomWithAlias)
			if err != nil {
				return err
			}
			return runReadSince(roomID, since, limit, jsonFlag, client.EventContext, client.MessagesFrom)
		}
		return runBoardRead(name, topic, aliasName, limit, jsonFlag, from, client.ResolveAlias, client.Messages, client.CreateRoomWithAlias)
	},
}

var boardWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch the board for new messages",
	Long:  "Block until a new message arrives on the board, print it, and exit. Use --follow to stream continuously.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		timeout, _ := cmd.Flags().GetInt("timeout")
		follow, _ := cmd.Flags().GetBool("follow")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		name, topic, aliasName, err := boardTarget(cmd)
		if err != nil {
			return err
		}
		return runBoardWatch(name, topic, aliasName, timeout, follow, client.ResolveAlias, client.Sync, client.CreateRoomWithAlias)
	},
}

func init() {
	boardCmd.PersistentFlags().Bool("global", false, "target the global construct board")
	boardReadCmd.Flags().IntP("limit", "n", 20, "number of messages to retrieve")
	boardReadCmd.Flags().Bool("json", false, "output messages as JSON")
	boardReadCmd.Flags().String("since", "", "show messages after this event ID")
	boardReadCmd.Flags().String("from", "", "filter messages by sender username")
	boardWatchCmd.Flags().Int("timeout", 30, "seconds to wait before giving up")
	boardWatchCmd.Flags().BoolP("follow", "f", false, "stream messages continuously")
	addCheckFlags(boardPostCmd)
	boardCmd.AddCommand(boardPostCmd)
	boardCmd.AddCommand(boardReadCmd)
	boardCmd.AddCommand(boardWatchCmd)
	Cmd.AddCommand(boardCmd)
}

// GlobalBoardAlias is the canonical alias name for the global construct board.
const GlobalBoardAlias = "construct-board"

// ProvisionGlobalBoard ensures the global construct board room exists and joins
// the current user to it. This is called during session provisioning so that
// every construct is in a shared room for cross-agent discovery.
func ProvisionGlobalBoard(token string) error {
	client := NewClient(Homeserver, token)
	roomID, err := ensureBoardRoom("construct-board", "Global construct board", GlobalBoardAlias, client.ResolveAlias, client.CreateRoomWithAlias)
	if err != nil {
		return fmt.Errorf("provisioning global board: %w", err)
	}
	// Join is idempotent — safe to call if already a member.
	if _, err := client.Join(roomID); err != nil {
		return fmt.Errorf("joining global board: %w", err)
	}
	return nil
}

// AnnounceOnBoard posts a message to the global construct board.
func AnnounceOnBoard(token, message string) error {
	client := NewClient(Homeserver, token)
	roomID, err := ensureBoardRoom("construct-board", "Global construct board", GlobalBoardAlias, client.ResolveAlias, client.CreateRoomWithAlias)
	if err != nil {
		return fmt.Errorf("resolving global board: %w", err)
	}
	if _, err := client.Send(roomID, message); err != nil {
		return fmt.Errorf("posting to global board: %w", err)
	}
	return nil
}

// boardTarget resolves the board name, topic, and alias based on --global flag.
func boardTarget(cmd *cobra.Command) (name, topic, aliasName string, err error) {
	global, _ := cmd.Flags().GetBool("global")
	if global {
		return "construct-board", "Global construct board", "construct-board", nil
	}
	agent, err := AgentFromEnv()
	if err != nil {
		return "", "", "", err
	}
	return "board-" + agent, fmt.Sprintf("Construct board for agent %s", agent), "board-" + agent, nil
}

func boardAlias(aliasName string) string {
	return "#" + aliasName + ":" + ServerName(Homeserver)
}

// ensureBoardRoom resolves the board room, creating it if it doesn't exist.
func ensureBoardRoom(name, topic, aliasName string, resolve AliasResolver, create func(name, topic, aliasName string) (*Room, error)) (string, error) {
	alias := boardAlias(aliasName)
	resp, err := resolve(alias)
	if err == nil {
		return resp.RoomID, nil
	}
	room, err := create(name, topic, aliasName)
	if err != nil {
		return "", fmt.Errorf("creating board room: %w", err)
	}
	return room.RoomID, nil
}

func runBoardPost(name, topic, aliasName, message string, resolve AliasResolver, send MessageSender, create func(string, string, string) (*Room, error)) error {
	roomID, err := ensureBoardRoom(name, topic, aliasName, resolve, create)
	if err != nil {
		return err
	}
	eventID, err := send(roomID, message)
	if err != nil {
		return err
	}
	fmt.Println(eventID)
	return nil
}

func runBoardRead(name, topic, aliasName string, limit int, jsonOut bool, from string, resolve AliasResolver, read MessageReader, create func(string, string, string) (*Room, error)) error {
	roomID, err := ensureBoardRoom(name, topic, aliasName, resolve, create)
	if err != nil {
		return err
	}
	if from != "" {
		return runReadFiltered(roomID, limit, jsonOut, from, read)
	}
	if jsonOut {
		return runReadJSON(roomID, limit, read)
	}
	return runRead(roomID, limit, read)
}

type syncFunc func(ctx context.Context, since string, timeout int, roomID string) (*SyncResponse, error)

func runBoardWatch(name, topic, aliasName string, timeout int, follow bool, resolve AliasResolver, sync syncFunc, create func(string, string, string) (*Room, error)) error {
	roomID, err := ensureBoardRoom(name, topic, aliasName, resolve, create)
	if err != nil {
		return err
	}

	ctx := context.Background()
	if !follow && timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second+5*time.Second)
		defer cancel()
	}

	// Initial sync to get the batch token.
	resp, err := sync(ctx, "", 0, roomID)
	if err != nil {
		return fmt.Errorf("initial sync: %w", err)
	}

	for {
		resp, err = sync(ctx, resp.NextBatch, pollInterval, roomID)
		if err != nil {
			if ctx.Err() != nil && !follow {
				return fmt.Errorf("no new messages within timeout")
			}
			return fmt.Errorf("sync: %w", err)
		}

		room, ok := resp.Rooms.Join[roomID]
		found := false
		if ok {
			for _, m := range room.Timeline.Events {
				if m.Type != msgTypeRoomMessage {
					continue
				}
				found = true
				body, _ := m.Content["body"].(string)
				fmt.Printf("%s: %s\n", m.Sender, body)
			}
		}

		if !follow {
			if found {
				return nil
			}
			if ctx.Err() != nil {
				return fmt.Errorf("no new messages within timeout")
			}
		}
	}
}
