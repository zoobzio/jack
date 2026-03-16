package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(whoamiCmd)
}

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Print the current session identity",
	Args:  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		token, err := TokenFromEnv()
		if err != nil {
			return err
		}
		client := NewClient(Homeserver, token)
		return runWhoAmI(client.WhoAmI)
	},
}

func runWhoAmI(whoami WhoAmIGetter) error {
	resp, err := whoami()
	if err != nil {
		return err
	}
	fmt.Println(resp.UserID)
	return nil
}
