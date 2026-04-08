package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ghExec runs a gh CLI command and returns the raw output.
func ghExec(args ...string) ([]byte, error) {
	cmd := exec.Command("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("gh %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("gh %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// ghJSON runs a gh CLI command and unmarshals the JSON output.
func ghJSON(result interface{}, args ...string) error {
	out, err := ghExec(args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, result); err != nil {
		return fmt.Errorf("decoding gh output: %w", err)
	}
	return nil
}

// ghWrite runs a gh CLI write command (comment, edit, label, etc.)
// and returns the raw output.
func ghWrite(args ...string) ([]byte, error) {
	return ghExec(args...)
}
