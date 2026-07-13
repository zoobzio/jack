package domain

import "testing"

func TestNewRepo(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    Repo
		wantErr bool
	}{
		{name: "scp style", url: "git@github.com:user/repo.git", want: "repo"},
		{name: "https with .git", url: "https://github.com/user/repo.git", want: "repo"},
		{name: "https without .git", url: "https://github.com/user/repo", want: "repo"},
		{name: "trailing .git stripped plain", url: "repo.git", want: "repo"},
		{name: "plain name", url: "repo", want: "repo"},
		{name: "hyphenated repo", url: "git@github.com:user/my-repo.git", want: "my-repo"},
		{name: "nested path", url: "https://host/group/sub/repo.git", want: "repo"},
		{name: "empty", url: "", wantErr: true},
		{name: "only .git", url: ".git", wantErr: true},
		{name: "scp trailing slash empty", url: "git@github.com:user/", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewRepo(tt.url)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewRepo(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("NewRepo(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestRepoValidate(t *testing.T) {
	tests := []struct {
		name    string
		repo    Repo
		wantErr bool
	}{
		{name: "empty", repo: "", wantErr: true},
		{name: "valid", repo: "repo", wantErr: false},
		{name: "valid hyphenated", repo: "my-repo", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.repo.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
