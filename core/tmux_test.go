package core

import "testing"

func TestNewTmuxNonNil(t *testing.T) {
	if NewTmux() == nil {
		t.Fatal("NewTmux returned nil")
	}
}

func TestTmuxParse(t *testing.T) {
	// name, created, activity, path, attached, windows
	output := "alpha\t1000000000\t1000000500\t/home/a\t1\t3\n" +
		"beta\t1000000100\t1000000200\t/home/b\t0\t1\n"

	tm := tmux{}
	got, err := tm.parse(output)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions, got %d: %+v", len(got), got)
	}

	if got[0].Name != "alpha" || got[0].Path != "/home/a" || !got[0].Attached || got[0].Windows != 3 {
		t.Errorf("session[0] = %+v", got[0])
	}
	if got[0].Created.Unix() != 1000000000 || got[0].Activity.Unix() != 1000000500 {
		t.Errorf("session[0] times = created %v activity %v", got[0].Created.Unix(), got[0].Activity.Unix())
	}
	if got[1].Name != "beta" || got[1].Attached || got[1].Windows != 1 {
		t.Errorf("session[1] = %+v", got[1])
	}
}

func TestTmuxParseEmpty(t *testing.T) {
	tm := tmux{}
	got, err := tm.parse("")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no sessions, got %+v", got)
	}
}

func TestTmuxParseMalformed(t *testing.T) {
	tm := tmux{}
	// Only three fields instead of six -> NewSession returns an error.
	if _, err := tm.parse("alpha\t1000000000\t1000000500\n"); err == nil {
		t.Fatal("expected error for malformed line, got nil")
	}
}

func TestTmuxParseBadTimestamp(t *testing.T) {
	tm := tmux{}
	if _, err := tm.parse("alpha\tnotanumber\t1000000500\t/home/a\t1\t3\n"); err == nil {
		t.Fatal("expected error for non-numeric timestamp, got nil")
	}
}
