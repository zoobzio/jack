package gh

import (
	"encoding/json"
	"fmt"
	"os"
)

// TaggedField wraps a user-authored string with classification metadata.
type TaggedField struct {
	Value          string   `json:"value"`
	Classification string   `json:"classification"`
	Score          float64  `json:"score,omitempty"`
	Flags          []string `json:"flags,omitempty"`
}

// Tag classifies a string and returns a tagged field.
// If the classifier is unavailable, the field is tagged as unclassified.
func Tag(classifier *Classifier, text, direction string) TaggedField {
	if classifier == nil || text == "" {
		return TaggedField{
			Value:          text,
			Classification: "unclassified",
		}
	}
	result, err := classifier.Classify(text, direction)
	if err != nil {
		return TaggedField{
			Value:          text,
			Classification: "error",
			Flags:          []string{err.Error()},
		}
	}
	return TaggedField{
		Value:          text,
		Classification: result.Classification,
		Score:          result.Score,
		Flags:          result.Flags,
	}
}

// TagComment classifies a comment body for the given direction.
type TaggedComment struct {
	Author TaggedField `json:"author"`
	Body   TaggedField `json:"body"`
}

// printJSON encodes a value as indented JSON to stdout.
func printJSON(v interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("encoding output: %w", err)
	}
	return nil
}
