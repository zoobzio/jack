//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunLoginSuccess(t *testing.T) {
	authenticator := func(user, pass string) (*Registration, error) {
		return &Registration{
			UserID:      "@" + user + ":localhost",
			AccessToken: "tok_login",
		}, nil
	}

	err := runLogin("operator", "pass", authenticator)
	jtesting.AssertNoError(t, err)
}

func TestRunLoginError(t *testing.T) {
	authenticator := func(_, _ string) (*Registration, error) {
		return nil, fmt.Errorf("bad credentials")
	}
	err := runLogin("x", "y", authenticator)
	jtesting.AssertError(t, err)
}
