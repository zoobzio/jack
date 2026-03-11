package msg

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	Cmd.AddCommand(registerCmd)
}

var registerCmd = &cobra.Command{
	Use:   "register <username> <password>",
	Short: "Register a Matrix user and print access token",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		client := NewClient(Homeserver, "")
		return runRegister(args[0], args[1], RegistrationToken, client.Register)
	},
}

func runRegister(username, password, regToken string, register Registerer) error {
	reg, err := register(username, password, regToken)
	if err != nil {
		return err
	}
	fmt.Printf("%s %s\n", reg.UserID, reg.AccessToken)
	return nil
}
