package domain

import (
	"strconv"
	"testing"
	"time"
)

func TestNewSession(t *testing.T) {
	now := time.Now()
	created := now.Add(-time.Hour)

	tests := []struct {
		name         string
		fields       []string
		wantErr      bool
		wantName     string
		wantAttached bool
		wantWindows  int
	}{
		{
			name: "valid attached",
			fields: []string{
				"sess",
				strconv.FormatInt(created.Unix(), 10),
				strconv.FormatInt(now.Unix(), 10),
				"/root/workspace/repo",
				"1",
				"3",
			},
			wantName:     "sess",
			wantAttached: true,
			wantWindows:  3,
		},
		{
			name: "valid not attached",
			fields: []string{
				"sess",
				strconv.FormatInt(created.Unix(), 10),
				strconv.FormatInt(now.Unix(), 10),
				"/root/workspace/repo",
				"0",
				"2",
			},
			wantName:     "sess",
			wantAttached: false,
			wantWindows:  2,
		},
		{
			name: "unparseable windows falls back to 1",
			fields: []string{
				"sess",
				strconv.FormatInt(created.Unix(), 10),
				strconv.FormatInt(now.Unix(), 10),
				"/root/workspace/repo",
				"0",
				"notanint",
			},
			wantName:     "sess",
			wantAttached: false,
			wantWindows:  1,
		},
		{
			name:    "too few fields",
			fields:  []string{"sess", "1", "2", "/path", "0"},
			wantErr: true,
		},
		{
			name:    "too many fields",
			fields:  []string{"sess", "1", "2", "/path", "0", "1", "extra"},
			wantErr: true,
		},
		{
			name: "unparseable created time",
			fields: []string{
				"sess", "notatime", strconv.FormatInt(now.Unix(), 10),
				"/path", "0", "1",
			},
			wantErr: true,
		},
		{
			name: "unparseable activity time",
			fields: []string{
				"sess", strconv.FormatInt(created.Unix(), 10), "notatime",
				"/path", "0", "1",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSession(tt.fields)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSession() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if s.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", s.Name, tt.wantName)
			}
			if s.Attached != tt.wantAttached {
				t.Errorf("Attached = %v, want %v", s.Attached, tt.wantAttached)
			}
			if s.Windows != tt.wantWindows {
				t.Errorf("Windows = %d, want %d", s.Windows, tt.wantWindows)
			}
		})
	}
}

func TestSessionStatus(t *testing.T) {
	tests := []struct {
		name    string
		session Session
		want    string
	}{
		{
			name:    "attached takes precedence",
			session: Session{Attached: true, Activity: time.Now().Add(-48 * time.Hour)},
			want:    "attached",
		},
		{
			name:    "active under a minute",
			session: Session{Attached: false, Activity: time.Now()},
			want:    "active",
		},
		{
			name:    "idle minutes",
			session: Session{Attached: false, Activity: time.Now().Add(-30 * time.Minute)},
			want:    "idle 30m",
		},
		{
			name:    "idle hours",
			session: Session{Attached: false, Activity: time.Now().Add(-3 * time.Hour)},
			want:    "idle 3h",
		},
		{
			name:    "idle days",
			session: Session{Attached: false, Activity: time.Now().Add(-50 * time.Hour)},
			want:    "idle 2d",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.Status(); got != tt.want {
				t.Fatalf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTimestamp(t *testing.T) {
	got, err := timestamp("  1700000000  ")
	if err != nil {
		t.Fatalf("timestamp() unexpected error: %v", err)
	}
	if want := time.Unix(1700000000, 0); !got.Equal(want) {
		t.Errorf("timestamp() = %v, want %v", got, want)
	}

	if _, err := timestamp("notanumber"); err == nil {
		t.Errorf("timestamp() error = nil, want error")
	}
}

func TestIntor(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		fallback int
		want     int
	}{
		{name: "parses int", in: "42", fallback: 1, want: 42},
		{name: "trims whitespace", in: "  7  ", fallback: 1, want: 7},
		{name: "fallback on garbage", in: "nope", fallback: 5, want: 5},
		{name: "fallback on empty", in: "", fallback: 9, want: 9},
		{name: "negative", in: "-3", fallback: 1, want: -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := intor(tt.in, tt.fallback); got != tt.want {
				t.Fatalf("intor(%q, %d) = %d, want %d", tt.in, tt.fallback, got, tt.want)
			}
		})
	}
}
