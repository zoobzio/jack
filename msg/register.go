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
	Short: "Register a Matrix user",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		client := NewClient(Homeserver, "")
		return runRegister(args[0], args[1], RegistrationToken, client.Register, SaveToken)
	},
}

func runRegister(username, password, regToken string, register Registerer, save TokenSaver) error {
	reg, err := register(username, password, regToken)
	if err != nil {
		return err
	}
	if err := save(username, reg.AccessToken); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}
	fmt.Printf("registered %s\n", reg.UserID)
	return nil
}
