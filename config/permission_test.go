package config

import "testing"

func TestPermissionValidate(t *testing.T) {
	for _, p := range []Permission{"", PermissionDefault, PermissionAcceptEdits, PermissionBypass} {
		if err := p.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", p, err)
		}
	}
	if err := Permission("yolo").Validate(); err == nil {
		t.Error("Validate(\"yolo\") = nil, want error")
	}
}

func TestPermissionFlags(t *testing.T) {
	tests := map[Permission]string{
		"":                    "",
		PermissionDefault:     "",
		PermissionAcceptEdits: "--permission-mode acceptEdits",
		PermissionBypass:      "--dangerously-skip-permissions",
	}
	for p, want := range tests {
		if got := p.Flags(); got != want {
			t.Errorf("Flags(%q) = %q, want %q", p, got, want)
		}
	}
}
