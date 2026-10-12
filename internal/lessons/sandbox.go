// Package lessons implements hands-on git exercises in throwaway repositories.
package lessons

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Sandbox is an isolated playground: a project folder plus room for fake
// remotes and teammates, with git configured independently of the user's
// own settings.
type Sandbox struct {
	Root    string // temp root holding everything
	Dir     string // the learner's project folder (Root/project)
	globalC string // isolated global git config file
}

// NewSandbox creates an empty sandbox on disk.
func NewSandbox() (*Sandbox, error) {
	root, err := os.MkdirTemp("", "gitwise-lesson-")
	if err != nil {
		return nil, err
	}
	sb := &Sandbox{Root: root, Dir: filepath.Join(root, "project"), globalC: filepath.Join(root, ".gitconfig")}
	if err := os.MkdirAll(sb.Dir, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(sb.globalC, nil, 0o644); err != nil {
		return nil, err
	}
	return sb, nil
}

// Remove deletes the sandbox.
func (sb *Sandbox) Remove() error { return os.RemoveAll(sb.Root) }

// Env is the environment every sandbox command runs with. It ignores the
// user's real git config (so hooks, signing or aliases can't interfere),
// never opens an editor or pager, and forces colored output.
func (sb *Sandbox) Env() []string {
	cfg := [][2]string{
		{"init.defaultBranch", "main"},
		{"user.name", "Gitwise Learner"},
		{"user.email", "learner@gitwise.local"},
		{"color.ui", "always"},
		{"pull.rebase", "false"},
		{"advice.detachedHead", "true"},
	}
	env := append(os.Environ(),
		"GIT_CONFIG_GLOBAL="+sb.globalC,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_EDITOR=true",
		"GIT_PAGER=cat",
		"PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT="+strconv.Itoa(len(cfg)),
	)
	for i, kv := range cfg {
		n := strconv.Itoa(i)
		env = append(env, "GIT_CONFIG_KEY_"+n+"="+kv[0], "GIT_CONFIG_VALUE_"+n+"="+kv[1])
	}
	return env
}

// Shell runs a command line in dir (relative to the sandbox root when not
// absolute) and returns its combined output.
func (sb *Sandbox) Shell(dir, line string) (string, error) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(sb.Root, dir)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", line)
	cmd.Dir = dir
	cmd.Env = sb.Env()
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out) + "\n(command timed out — interactive programs are not supported here)", ctx.Err()
	}
	return string(out), err
}

// Git runs git in the project folder and returns trimmed output.
func (sb *Sandbox) Git(args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = sb.Dir
	cmd.Env = sb.Env()
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

// GitOK runs git in the project folder and reports success.
func (sb *Sandbox) GitOK(args ...string) bool {
	cmd := exec.Command("git", args...)
	cmd.Dir = sb.Dir
	cmd.Env = sb.Env()
	return cmd.Run() == nil
}

// script runs setup commands in order, stopping at the first failure.
func (sb *Sandbox) script(dir string, lines ...string) error {
	for _, l := range lines {
		if out, err := sb.Shell(dir, l); err != nil {
			return &setupError{line: l, out: out, err: err}
		}
	}
	return nil
}

type setupError struct {
	line, out string
	err       error
}

func (e *setupError) Error() string {
	return "lesson setup failed at `" + e.line + "`: " + e.err.Error() + "\n" + e.out
}

// --- helpers used by step checks ---

func (sb *Sandbox) exists(name string) bool {
	_, err := os.Stat(filepath.Join(sb.Dir, name))
	return err == nil
}

func (sb *Sandbox) read(name string) string {
	b, _ := os.ReadFile(filepath.Join(sb.Dir, name))
	return string(b)
}

func (sb *Sandbox) commitCount(rev string) int {
	n, _ := strconv.Atoi(sb.Git("rev-list", "--count", rev))
	return n
}

func (sb *Sandbox) branch() string { return sb.Git("symbolic-ref", "--short", "-q", "HEAD") }

func (sb *Sandbox) isStaged(path string) bool {
	return containsLine(sb.Git("diff", "--cached", "--name-only"), path)
}

func (sb *Sandbox) isDirty(path string) bool {
	return sb.Git("status", "--porcelain", "--", path) != ""
}

func (sb *Sandbox) stashCount() int {
	out := sb.Git("stash", "list")
	if out == "" {
		return 0
	}
	return len(strings.Split(out, "\n"))
}

func (sb *Sandbox) inMerge() bool { return sb.GitOK("rev-parse", "-q", "--verify", "MERGE_HEAD") }

func containsLine(out, want string) bool {
	for _, l := range strings.Split(out, "\n") {
		if strings.TrimSpace(l) == want {
			return true
		}
	}
	return false
}
