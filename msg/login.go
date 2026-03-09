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
	Short: "Login and store access token",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		client := NewClient(Homeserver, "")
		return runLogin(args[0], args[1], client.Login, SaveToken)
	},
}

func runLogin(username, password string, login Authenticator, save TokenSaver) error {
	reg, err := login(username, password)
	if err != nil {
		return err
	}
	if err := save(username, reg.AccessToken); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	fmt.Printf("logged in as %s\n", reg.UserID)
	return nil
}
