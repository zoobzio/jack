package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login <username> <password>",
	Short: "Login and print access token",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		client := NewClient(Homeserver, "")
		return runLogin(args[0], args[1], client.Login)
	},
}

func runLogin(username, password string, login Authenticator) error {
	reg, err := login(username, password)
	if err != nil {
		return err
	}
	fmt.Printf("%s %s\n", reg.UserID, reg.AccessToken)
	return nil
}
