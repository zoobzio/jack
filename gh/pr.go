package gh

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Classified GitHub PR operations",
}

var prViewCmd = &cobra.Command{
	Use:   "view <number>",
	Short: "View a PR with classified content",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		num, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %s", args[0])
		}
		return runPRView(num)
	},
}

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List PRs with classified titles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		state, _ := cmd.Flags().GetString("state")
		label, _ := cmd.Flags().GetString("label")
		author, _ := cmd.Flags().GetString("author")
		return runPRList(limit, state, label, author)
	},
}

var prCommentCmd = &cobra.Command{
	Use:   "comment <number> --body <message>",
	Short: "Comment on a PR (write-path classified)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		body, _ := cmd.Flags().GetString("body")
		if body == "" {
			return fmt.Errorf("--body is required")
		}
		return runPRComment(args[0], body)
	},
}

func init() {
	prListCmd.Flags().IntP("limit", "n", 20, "number of PRs to list")
	prListCmd.Flags().String("state", "open", "PR state filter")
	prListCmd.Flags().StringP("label", "l", "", "filter by label")
	prListCmd.Flags().String("author", "", "filter by author")
	prCommentCmd.Flags().String("body", "", "comment body")
	prCmd.AddCommand(prViewCmd)
	prCmd.AddCommand(prListCmd)
	prCmd.AddCommand(prCommentCmd)
	Cmd.AddCommand(prCmd)
}

// --- raw gh JSON types ---

type ghPR struct {
	Title    string      `json:"title"`
	Body     string      `json:"body"`
	State    string      `json:"state"`
	Author   ghUser      `json:"author"`
	Labels   []ghLabel   `json:"labels"`
	Comments []ghComment `json:"comments"`
	Reviews  []ghReview  `json:"reviews"`
	Number   int         `json:"number"`
}

type ghReview struct {
	Author ghUser `json:"author"`
	Body   string `json:"body"`
	State  string `json:"state"`
}

// --- classified output types ---

type classifiedPR struct {
	State    string             `json:"state"`
	Author   string             `json:"author"`
	Title    TaggedField        `json:"title"`
	Body     TaggedField        `json:"body"`
	Labels   []string           `json:"labels"`
	Comments []TaggedComment    `json:"comments"`
	Reviews  []classifiedReview `json:"reviews"`
	Number   int                `json:"number"`
}

type classifiedReview struct {
	Author string      `json:"author"`
	State  string      `json:"state"`
	Body   TaggedField `json:"body"`
}

type classifiedPRSummary struct {
	State  string      `json:"state"`
	Author string      `json:"author"`
	Title  TaggedField `json:"title"`
	Labels []string    `json:"labels"`
	Number int         `json:"number"`
}

// --- implementation ---

func runPRView(number int) error {
	var pr ghPR
	if err := ghJSON(&pr, "pr", "view", strconv.Itoa(number),
		"--json", "number,title,body,state,labels,author,comments,reviews"); err != nil {
		return err
	}

	classifier := classifierInstance()
	labels := make([]string, len(pr.Labels))
	for i, l := range pr.Labels {
		labels[i] = l.Name
	}

	comments := make([]TaggedComment, len(pr.Comments))
	for i, c := range pr.Comments {
		comments[i] = TaggedComment{
			Author: Tag(classifier, c.Author.Login, "read"),
			Body:   Tag(classifier, c.Body, "read"),
		}
	}

	reviews := make([]classifiedReview, len(pr.Reviews))
	for i, r := range pr.Reviews {
		reviews[i] = classifiedReview{
			Author: r.Author.Login,
			Body:   Tag(classifier, r.Body, "read"),
			State:  r.State,
		}
	}

	out := classifiedPR{
		Number:   pr.Number,
		Title:    Tag(classifier, pr.Title, "read"),
		Body:     Tag(classifier, pr.Body, "read"),
		State:    pr.State,
		Labels:   labels,
		Author:   pr.Author.Login,
		Comments: comments,
		Reviews:  reviews,
	}
	return printJSON(out)
}

func runPRList(limit int, state, label, author string) error {
	args := []string{"pr", "list",
		"--state", state,
		"--limit", strconv.Itoa(limit),
		"--json", "number,title,state,labels,author",
	}
	if label != "" {
		args = append(args, "--label", label)
	}
	if author != "" {
		args = append(args, "--author", author)
	}

	var prs []ghPR
	if err := ghJSON(&prs, args...); err != nil {
		return err
	}

	classifier := classifierInstance()
	out := make([]classifiedPRSummary, len(prs))
	for i, pr := range prs {
		labels := make([]string, len(pr.Labels))
		for j, l := range pr.Labels {
			labels[j] = l.Name
		}
		out[i] = classifiedPRSummary{
			Number: pr.Number,
			Title:  Tag(classifier, pr.Title, "read"),
			State:  pr.State,
			Labels: labels,
			Author: pr.Author.Login,
		}
	}
	return printJSON(out)
}

func runPRComment(number, body string) error {
	classifier := classifierInstance()
	if classifier != nil {
		result := Tag(classifier, body, "write")
		if result.Classification == "suspicious" {
			return fmt.Errorf("write blocked: outgoing comment classified as suspicious (score=%.4f, flags=%v)", result.Score, result.Flags)
		}
	}
	_, err := ghWrite("pr", "comment", number, "--body", body)
	return err
}
