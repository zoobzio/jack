package msg

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const msgTypeReaction = "m.reaction"
const doneReactionKey = "\u2705"

var noteCmd = &cobra.Command{
	Use:   "note",
	Short: "Personal reminders",
	Long:  "Add, list, and resolve personal reminder notes.",
}

var noteAddCmd = &cobra.Command{
	Use:   "add <message...>",
	Short: "Add a reminder note",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		message := strings.Join(args, " ")
		name, topic, aliasName, err := noteTarget()
		if err != nil {
			return err
		}
		if err := runNoteAdd(name, topic, aliasName, message, client.ResolveAlias, client.Send, client.CreateRoomWithAlias); err != nil {
			return err
		}
		return postCheck(cmd)
	},
}

var noteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending notes",
	Long:  "List pending reminder notes. Use --all to include resolved notes.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		all, _ := cmd.Flags().GetBool("all")
		limit, _ := cmd.Flags().GetInt("limit")
		jsonFlag, _ := cmd.Flags().GetBool("json")
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		name, topic, aliasName, err := noteTarget()
		if err != nil {
			return err
		}
		return runNoteList(name, topic, aliasName, limit, all, jsonFlag, client.ResolveAlias, client.Messages, client.CreateRoomWithAlias)
	},
}

var noteDoneCmd = &cobra.Command{
	Use:   "done <event_id>",
	Short: "Mark a note as done",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		name, topic, aliasName, err := noteTarget()
		if err != nil {
			return err
		}
		return runNoteDone(name, topic, aliasName, args[0], client.ResolveAlias, client.SendReaction, client.CreateRoomWithAlias)
	},
}

func init() {
	noteListCmd.Flags().BoolP("all", "a", false, "include resolved notes")
	noteListCmd.Flags().IntP("limit", "n", 50, "number of events to retrieve")
	noteListCmd.Flags().Bool("json", false, "output notes as JSON")
	addCheckFlags(noteAddCmd)
	noteCmd.AddCommand(noteAddCmd)
	noteCmd.AddCommand(noteListCmd)
	noteCmd.AddCommand(noteDoneCmd)
	Cmd.AddCommand(noteCmd)
}

func noteTarget() (name, topic, aliasName string, err error) {
	agent, err := AgentFromEnv()
	if err != nil {
		return "", "", "", err
	}
	return "note-" + agent, fmt.Sprintf("Notes for agent %s", agent), "note-" + agent, nil
}

func runNoteAdd(name, topic, aliasName, message string, resolve AliasResolver, send MessageSender, create func(string, string, string) (*Room, error)) error {
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

type noteEntry struct {
	EventID string `json:"event_id"`
	Body    string `json:"body"`
	Done    bool   `json:"done"`
}

func runNoteList(name, topic, aliasName string, limit int, all, jsonOut bool, resolve AliasResolver, read MessageReader, create func(string, string, string) (*Room, error)) error {
	roomID, err := ensureBoardRoom(name, topic, aliasName, resolve, create)
	if err != nil {
		return err
	}
	msgs, err := read(roomID, limit)
	if err != nil {
		return err
	}

	// Build set of event IDs that have been marked done via reaction.
	done := make(map[string]bool)
	for _, m := range msgs.Chunk {
		if m.Type != msgTypeReaction {
			continue
		}
		rel, ok := m.Content["m.relates_to"].(map[string]interface{})
		if !ok {
			continue
		}
		key, _ := rel["key"].(string)
		target, _ := rel["event_id"].(string)
		if key == doneReactionKey && target != "" {
			done[target] = true
		}
	}

	// Collect notes (messages) in chronological order (chunk is reverse).
	var notes []noteEntry
	for i := len(msgs.Chunk) - 1; i >= 0; i-- {
		m := msgs.Chunk[i]
		if m.Type != msgTypeRoomMessage {
			continue
		}
		body, _ := m.Content["body"].(string)
		entry := noteEntry{
			EventID: m.EventID,
			Body:    body,
			Done:    done[m.EventID],
		}
		if !all && entry.Done {
			continue
		}
		notes = append(notes, entry)
	}

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(notes)
	}

	for _, n := range notes {
		mark := "[ ]"
		if n.Done {
			mark = "[x]"
		}
		fmt.Printf("%s %s  %s\n", mark, n.EventID, n.Body)
	}
	return nil
}

func runNoteDone(name, topic, aliasName, eventID string, resolve AliasResolver, react ReactionSender, create func(string, string, string) (*Room, error)) error {
	roomID, err := ensureBoardRoom(name, topic, aliasName, resolve, create)
	if err != nil {
		return err
	}
	_, err = react(roomID, eventID, doneReactionKey)
	return err
}
