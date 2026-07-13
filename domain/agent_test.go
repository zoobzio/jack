package domain

import "testing"

func TestAgentValidate(t *testing.T) {
	tests := []struct {
		name    string
		agent   Agent
		wantErr bool
	}{
		{name: "empty", agent: "", wantErr: true},
		{name: "contains hyphen", agent: "foo-bar", wantErr: true},
		{name: "leading hyphen", agent: "-foo", wantErr: true},
		{name: "trailing hyphen", agent: "foo-", wantErr: true},
		{name: "valid simple", agent: "claude", wantErr: false},
		{name: "valid with underscore", agent: "my_agent", wantErr: false},
		{name: "valid alphanumeric", agent: "agent123", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.agent.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
