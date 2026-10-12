package git

import (
	"os"
	"path/filepath"
	"testing"
)

func sh(t *testing.T, dir string, args ...string) {
	t.Helper()
	env := []string{"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t"}
	if _, err := RunEnv(dir, env, args...); err != nil {
		t.Fatal(err)
	}
}

func TestGetStatus(t *testing.T) {
	dir := t.TempDir()
	if _, err := GetStatus(dir); err != ErrNotRepo {
		t.Fatalf("want ErrNotRepo, got %v", err)
	}
	sh(t, dir, "init", "-q", "-b", "main")
	s, err := GetStatus(dir)
	if err != nil || !s.Initial || s.Branch != "main" || !s.Clean() {
		t.Fatalf("fresh repo: %+v %v", s, err)
	}

	write := func(name, body string) { os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644) }
	write("a.txt", "a")
	write("b c.txt", "b")
	sh(t, dir, "add", ".")
	sh(t, dir, "commit", "-qm", "init")
	write("a.txt", "changed")
	sh(t, dir, "mv", "b c.txt", "d.txt")
	write("new.txt", "n")

	s, err = GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if s.Initial || len(s.Unstaged) != 1 || s.Unstaged[0].Path != "a.txt" {
		t.Errorf("unstaged: %+v", s.Unstaged)
	}
	if len(s.Staged) != 1 || s.Staged[0].Code != 'R' || s.Staged[0].OrigPath != "b c.txt" || s.Staged[0].Path != "d.txt" {
		t.Errorf("staged rename: %+v", s.Staged)
	}
	if len(s.Untracked) != 1 || s.Untracked[0] != "new.txt" {
		t.Errorf("untracked: %+v", s.Untracked)
	}
	if s.LastCommit == "" {
		t.Error("missing last commit")
	}
}
