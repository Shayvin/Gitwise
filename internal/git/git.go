// Package git wraps the git binary and parses its machine-readable output.
package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ErrNotRepo is returned when an operation needs a repository but none is found.
var ErrNotRepo = errors.New("not inside a git repository")

// Run executes git with args in dir and returns trimmed stdout.
func Run(dir string, args ...string) (string, error) {
	return RunEnv(dir, nil, args...)
}

// RunEnv is like Run but appends extra environment variables.
func RunEnv(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if env != nil {
		cmd.Env = append(os.Environ(), env...)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		out := strings.TrimRight(stdout.String(), "\n")
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return out, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// Installed reports whether a git binary is on PATH.
func Installed() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Version returns the installed git version, e.g. "2.45.1".
func Version() (string, error) {
	out, err := Run("", "--version")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) < 3 {
		return out, nil
	}
	return fields[2], nil
}

// IsRepo reports whether dir is inside a git work tree.
func IsRepo(dir string) bool {
	out, err := Run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// Config returns the effective value of a config key, or "" if unset.
func Config(dir, key string) string {
	out, _ := Run(dir, "config", "--get", key)
	return out
}

// CurrentBranch returns the checked-out branch name, or "" when detached.
func CurrentBranch(dir string) string {
	out, err := Run(dir, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// RefExists reports whether rev resolves to an object.
func RefExists(dir, rev string) bool {
	_, err := Run(dir, "rev-parse", "--verify", "--quiet", rev+"^{object}")
	return err == nil
}

// GitDir returns the absolute path of the repository's .git directory.
func GitDir(dir string) (string, error) {
	return Run(dir, "rev-parse", "--absolute-git-dir")
}

// TopLevel returns the root of the work tree.
func TopLevel(dir string) (string, error) {
	return Run(dir, "rev-parse", "--show-toplevel")
}

// Change is a single changed path in the index or work tree.
type Change struct {
	Path     string
	OrigPath string // set for renames and copies
	Code     byte   // porcelain status letter: M, A, D, R, C, T
}

// Describe returns a human label for the change kind.
func (c Change) Describe() string {
	switch c.Code {
	case 'M':
		return "modified"
	case 'A':
		return "new file"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'T':
		return "type changed"
	}
	return "changed"
}

// Label returns the path, showing the origin for renames.
func (c Change) Label() string {
	if c.OrigPath != "" {
		return c.OrigPath + " → " + c.Path
	}
	return c.Path
}

// Status is a snapshot of the repository state.
type Status struct {
	Branch     string
	Detached   bool
	Initial    bool // no commits yet
	Upstream   string
	Ahead      int
	Behind     int
	Staged     []Change
	Unstaged   []Change
	Untracked  []string
	Conflicted []string
	Stashes    int
	Operation  string // merge, rebase, cherry-pick, revert, bisect or ""
	LastCommit string
}

// Clean reports whether there is nothing to commit and no untracked files.
func (s *Status) Clean() bool {
	return len(s.Staged)+len(s.Unstaged)+len(s.Untracked)+len(s.Conflicted) == 0
}

// GetStatus inspects the repository containing dir.
func GetStatus(dir string) (*Status, error) {
	if !IsRepo(dir) {
		return nil, ErrNotRepo
	}
	out, err := Run(dir, "status", "--porcelain=v2", "--branch", "-z")
	if err != nil {
		return nil, err
	}

	s := &Status{}
	entries := strings.Split(out, "\x00")
	for i := 0; i < len(entries); i++ {
		e := entries[i]
		if len(e) < 2 {
			continue
		}
		switch e[0] {
		case '#':
			parseHeader(s, e)
		case '1':
			if f := strings.SplitN(e, " ", 9); len(f) == 9 {
				s.addXY(f[1], f[8], "")
			}
		case '2':
			if f := strings.SplitN(e, " ", 10); len(f) == 10 {
				orig := ""
				if i+1 < len(entries) {
					i++
					orig = entries[i]
				}
				s.addXY(f[1], f[9], orig)
			}
		case 'u':
			if f := strings.SplitN(e, " ", 11); len(f) == 11 {
				s.Conflicted = append(s.Conflicted, f[10])
			}
		case '?':
			s.Untracked = append(s.Untracked, e[2:])
		}
	}

	if stashes, err := Run(dir, "stash", "list"); err == nil && stashes != "" {
		s.Stashes = len(strings.Split(stashes, "\n"))
	}
	if gitDir, err := GitDir(dir); err == nil {
		s.Operation = detectOperation(gitDir)
	}
	if !s.Initial {
		s.LastCommit, _ = Run(dir, "log", "-1", "--format=%h %s (%cr)")
	}
	return s, nil
}

func parseHeader(s *Status, line string) {
	f := strings.Fields(line)
	if len(f) < 3 {
		return
	}
	switch f[1] {
	case "branch.oid":
		s.Initial = f[2] == "(initial)"
	case "branch.head":
		if f[2] == "(detached)" {
			s.Detached = true
		} else {
			s.Branch = f[2]
		}
	case "branch.upstream":
		s.Upstream = f[2]
	case "branch.ab":
		if len(f) >= 4 {
			fmt.Sscanf(f[2], "+%d", &s.Ahead)
			fmt.Sscanf(f[3], "-%d", &s.Behind)
		}
	}
}

func (s *Status) addXY(xy, path, orig string) {
	if len(xy) != 2 {
		return
	}
	if xy[0] != '.' {
		s.Staged = append(s.Staged, Change{Path: path, OrigPath: orig, Code: xy[0]})
	}
	if xy[1] != '.' {
		s.Unstaged = append(s.Unstaged, Change{Path: path, Code: xy[1]})
	}
}

func detectOperation(gitDir string) string {
	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(gitDir, name))
		return err == nil
	}
	switch {
	case exists("rebase-merge"), exists("rebase-apply"):
		return "rebase"
	case exists("MERGE_HEAD"):
		return "merge"
	case exists("CHERRY_PICK_HEAD"):
		return "cherry-pick"
	case exists("REVERT_HEAD"):
		return "revert"
	case exists("BISECT_LOG"):
		return "bisect"
	}
	return ""
}
