//go:build testing

package jack

import (
	"testing"

	jtesting "github.com/zoobzio/jack/testing"
)

func TestTokenAgePath(t *testing.T) {
	jtesting.AssertEqual(t, tokenAgePath("/home/user/.jack/blue/vicky"), "/home/user/.jack/blue/vicky/.jack/token.age")
}

func TestDescriptionPath(t *testing.T) {
	jtesting.AssertEqual(t, descriptionPath("/home/user/.jack/blue/vicky"), "/home/user/.jack/blue/vicky/.jack/description.txt")
}
