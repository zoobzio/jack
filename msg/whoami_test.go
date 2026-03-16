//go:build testing

package msg

import (
	"fmt"
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestRunWhoAmISuccess(t *testing.T) {
	getter := func() (*WhoAmIResponse, error) {
		return &WhoAmIResponse{UserID: "@blue-vicky:localhost"}, nil
	}
	err := runWhoAmI(getter)
	jtesting.AssertNoError(t, err)
}

func TestRunWhoAmIError(t *testing.T) {
	getter := func() (*WhoAmIResponse, error) {
		return nil, fmt.Errorf("unauthorized")
	}
	err := runWhoAmI(getter)
	jtesting.AssertError(t, err)
}
