package gh

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Classified GitHub issue operations",
}

var issueViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View an issue with classified content",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %s", args[0])
		}
		return runIssueView(num)
	},
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues with classified titles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		state, _ := cmd.Flags().GetString("state")
		label, _ := cmd.Flags().GetString("label")
		return runIssueList(limit, state, label)
	},
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment <number> --body <message>",
	Short: "Comment on an issue (write-path classified)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, _ := cmd.Flags().GetString("body")
		if body == "" {
			return fmt.Errorf("--body is required")
		}
		return runIssueComment(args[0], body)
	},
}

func init() {
	issueListCmd.Flags().IntP("limit", "n", 20, "number of issues to list")
	issueListCmd.Flags().String("state", "open", "issue state filter")
	issueListCmd.Flags().StringP("label", "l", "", "filter by label")
	issueCommentCmd.Flags().String("body", "", "comment body")
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueCommentCmd)
	Cmd.AddCommand(issueCmd)
}

// --- raw gh JSON types ---

type ghIssue struct {
	Number   int         `json:"number"`
	Title    string      `json:"title"`
	Body     string      `json:"body"`
	State    string      `json:"state"`
	Labels   []ghLabel   `json:"labels"`
	Author   ghUser      `json:"author"`
	Comments []ghComment `json:"comments"`
}

type ghLabel struct {
	Name string `json:"name"`
}

type ghUser struct {
	Login string `json:"login"`
}

type ghComment struct {
	Author ghUser `json:"author"`
	Body   string `json:"body"`
}

// --- classified output types ---

type classifiedIssue struct {
	Number   int              `json:"number"`
	Title    TaggedField      `json:"title"`
	Body     TaggedField      `json:"body"`
	State    string           `json:"state"`
	Labels   []string         `json:"labels"`
	Author   string           `json:"author"`
	Comments []TaggedComment  `json:"comments"`
}

type classifiedIssueSummary struct {
	Number int         `json:"number"`
	Title  TaggedField `json:"title"`
	State  string      `json:"state"`
	Labels []string    `json:"labels"`
	Author string      `json:"author"`
}

// --- implementation ---

func runIssueView(number int) error {
	var issue ghIssue
	if err := ghJSON(&issue, "issue", "view", strconv.Itoa(number),
		"--json", "number,title,body,state,labels,author,comments"); err != nil {
		return err
	}

	classifier := classifierInstance()
	labels := make([]string, len(issue.Labels))
	for i, l := range issue.Labels {
		labels[i] = l.Name
	}

	comments := make([]TaggedComment, len(issue.Comments))
	for i, c := range issue.Comments {
		comments[i] = TaggedComment{
			Author: Tag(classifier, c.Author.Login, "read"),
			Body:   Tag(classifier, c.Body, "read"),
		}
	}

	out := classifiedIssue{
		Number:   issue.Number,
		Title:    Tag(classifier, issue.Title, "read"),
		Body:     Tag(classifier, issue.Body, "read"),
		State:    issue.State,
		Labels:   labels,
		Author:   issue.Author.Login,
		Comments: comments,
	}
	return printJSON(out)
}

func runIssueList(limit int, state, label string) error {
	args := []string{"issue", "list",
		"--state", state,
		"--limit", strconv.Itoa(limit),
		"--json", "number,title,state,labels,author",
	}
	if label != "" {
		args = append(args, "--label", label)
	}

	var issues []ghIssue
	if err := ghJSON(&issues, args...); err != nil {
		return err
	}

	classifier := classifierInstance()
	out := make([]classifiedIssueSummary, len(issues))
	for i, issue := range issues {
		labels := make([]string, len(issue.Labels))
		for j, l := range issue.Labels {
			labels[j] = l.Name
		}
		out[i] = classifiedIssueSummary{
			Number: issue.Number,
			Title:  Tag(classifier, issue.Title, "read"),
			State:  issue.State,
			Labels: labels,
			Author: issue.Author.Login,
		}
	}
	return printJSON(out)
}

func runIssueComment(number, body string) error {
	classifier := classifierInstance()
	if classifier != nil {
		result := Tag(classifier, body, "write")
		if result.Classification == "suspicious" {
			return fmt.Errorf("write blocked: outgoing comment classified as suspicious (score=%.4f, flags=%v)", result.Score, result.Flags)
		}
	}
	_, err := ghWrite("issue", "comment", number, "--body", body)
	return err
}
