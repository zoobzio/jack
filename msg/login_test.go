//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunLoginSuccess(t *testing.T) {
	var savedUser, savedToken string
	saver := func(user, token string) error {
		savedUser = user
		savedToken = token
		return nil
	}
	authenticator := func(user, pass string) (*Registration, error) {
		return &Registration{
			UserID:      "@" + user + ":localhost",
			AccessToken: "tok_login",
		}, nil
	}

	err := runLogin("operator", "pass", authenticator, saver)
	jtesting.AssertNoError(t, err)
	jtesting.AssertEqual(t, savedUser, "operator")
	jtesting.AssertEqual(t, savedToken, "tok_login")
}

func TestRunLoginError(t *testing.T) {
	authenticator := func(_, _ string) (*Registration, error) {
		return nil, fmt.Errorf("bad credentials")
	}
	err := runLogin("x", "y", authenticator, noopSaver)
	jtesting.AssertError(t, err)
}
