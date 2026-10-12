package knowledge

import (
	"fmt"
	"strings"
	"time"
)

// Risk rates how much damage a command can do.
type Risk int

const (
	RiskNone Risk = iota
	RiskLow
	RiskMedium
	RiskHigh
)

func (r Risk) String() string {
	switch r {
	case RiskLow:
		return "low risk"
	case RiskMedium:
		return "medium risk"
	case RiskHigh:
		return "high risk"
	}
	return "safe"
}

// Warning explains why a command is dangerous and what to do instead.
type Warning struct {
	Risk        Risk
	Title       string
	Explanation string
	Safer       string   // a safer alternative command, if any
	Backup      []string // git args that create a safety net before running
}

// SplitArgs strips a leading "git" and global options and returns the
// subcommand and its arguments.
func SplitArgs(args []string) (sub string, rest []string) {
	if len(args) > 0 && args[0] == "git" {
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-C" || a == "-c" || a == "--git-dir" || a == "--work-tree":
			i++ // option with a separate value
		case strings.HasPrefix(a, "-"):
		default:
			return a, args[i+1:]
		}
	}
	return "", nil
}

// hasFlag reports whether any of the long/short names appear in args.
// Single-letter short names also match inside combined flags like -fd.
func hasFlag(args []string, names ...string) bool {
	for _, a := range args {
		if a == "--" {
			return false
		}
		name, _, _ := strings.Cut(a, "=")
		for _, n := range names {
			if a == n || name == n {
				return true
			}
			if len(n) == 2 && n[0] == '-' && n[1] != '-' &&
				len(a) > 2 && a[0] == '-' && a[1] != '-' && strings.ContainsRune(a[1:], rune(n[1])) {
				return true
			}
		}
	}
	return false
}

func positional(args []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			out = append(out, a)
		}
	}
	return out
}

func backupBranch() []string {
	return []string{"branch", "gitwise-backup/" + time.Now().Format("20060102-150405")}
}

// Analyze inspects a git invocation and returns warnings, most severe first.
func Analyze(args []string) []Warning {
	sub, rest := SplitArgs(args)
	var w []Warning
	switch sub {
	case "push":
		switch {
		case hasFlag(rest, "--force", "-f"):
			w = append(w, Warning{
				Risk:        RiskHigh,
				Title:       "Force push overwrites the remote branch",
				Explanation: "Any commits on the server that you don't have locally are deleted — including your teammates' work.",
				Safer:       "git push --force-with-lease",
			})
		case hasFlag(rest, "--force-with-lease"):
			w = append(w, Warning{
				Risk:        RiskMedium,
				Title:       "Rewriting remote history",
				Explanation: "Safer than --force because it aborts if someone else pushed, but anyone who already pulled your old commits will have to fix their copy.",
			})
		}
		for _, p := range positional(rest) {
			if strings.HasPrefix(p, "+") {
				w = append(w, Warning{Risk: RiskHigh, Title: "A '+' refspec is a force push", Explanation: fmt.Sprintf("%q force-updates the remote branch.", p)})
			}
			if strings.HasPrefix(p, ":") {
				w = append(w, Warning{Risk: RiskMedium, Title: "This deletes a remote branch", Explanation: fmt.Sprintf("%q pushes 'nothing' to the branch, deleting it on the server.", p)})
			}
		}
		if hasFlag(rest, "--delete", "-d") {
			w = append(w, Warning{Risk: RiskMedium, Title: "Deleting a branch on the remote", Explanation: "Other people may still be using it."})
		}
	case "reset":
		if hasFlag(rest, "--hard") {
			w = append(w, Warning{
				Risk:        RiskHigh,
				Title:       "reset --hard discards uncommitted work",
				Explanation: "All staged and unstaged changes to tracked files are lost permanently. Committed work can still be recovered with `git reflog`.",
				Safer:       "git stash   (then reset; get changes back with git stash pop)",
				Backup:      backupBranch(),
			})
		} else if len(positional(rest)) > 0 && !hasFlag(rest, "--soft", "--mixed") && !contains(rest, "--") {
			w = append(w, Warning{Risk: RiskLow, Title: "Moves your branch or unstages files", Explanation: "Your files are kept, but if you name a commit, the branch label moves back and later commits disappear from `git log` (they stay in `git reflog`)."})
		}
	case "clean":
		if hasFlag(rest, "-f", "--force") && !hasFlag(rest, "-n", "--dry-run") {
			w = append(w, Warning{
				Risk:        RiskHigh,
				Title:       "clean permanently deletes untracked files",
				Explanation: "Files Git has never seen cannot be recovered — not by Git, not from the trash.",
				Safer:       "git clean -n   (preview first)",
			})
			if hasFlag(rest, "-x") {
				w = append(w, Warning{Risk: RiskHigh, Title: "-x also deletes ignored files", Explanation: "That includes things like .env files, local config and build caches."})
			}
		}
	case "checkout":
		if contains(rest, "--") || contains(positional(rest), ".") {
			w = append(w, Warning{
				Risk:        RiskHigh,
				Title:       "checkout on files throws away your edits",
				Explanation: "The files are overwritten with the committed version without asking.",
				Safer:       "git stash   (keeps a copy you can bring back)",
			})
		}
	case "restore":
		if !hasFlag(rest, "--staged", "-S") || hasFlag(rest, "--worktree", "-W") {
			w = append(w, Warning{
				Risk:        RiskHigh,
				Title:       "restore overwrites your files",
				Explanation: "Uncommitted edits in the given files are lost permanently.",
				Safer:       "git restore --staged <file>   (if you only want to unstage)",
			})
		}
	case "branch":
		if hasFlag(rest, "-D") || (hasFlag(rest, "--delete", "-d") && hasFlag(rest, "--force", "-f")) {
			w = append(w, Warning{
				Risk:        RiskMedium,
				Title:       "Force-deleting a branch",
				Explanation: "Commits that exist only on this branch become unreachable. They can be recovered from `git reflog` for a while.",
				Safer:       "git branch -d <name>   (refuses if unmerged)",
			})
		}
	case "rebase":
		if !hasFlag(rest, "--abort", "--continue", "--skip", "--quit") {
			w = append(w, Warning{
				Risk:        RiskMedium,
				Title:       "Rebase rewrites commits",
				Explanation: "Your commits are replaced by new copies. Never rebase commits that are already pushed and shared with others.",
				Backup:      backupBranch(),
			})
		}
	case "commit":
		if hasFlag(rest, "--amend") {
			w = append(w, Warning{Risk: RiskLow, Title: "Amend replaces the last commit", Explanation: "Fine for local commits. If you already pushed it, you'd need a force push afterwards."})
		}
	case "stash":
		if len(rest) > 0 && (rest[0] == "drop" || rest[0] == "clear") {
			w = append(w, Warning{Risk: RiskMedium, Title: "Deleting stashed changes", Explanation: "Dropped stashes are very hard to recover."})
		}
	case "rm":
		if !hasFlag(rest, "--cached") {
			w = append(w, Warning{Risk: RiskLow, Title: "rm also deletes the file from disk", Explanation: "Use --cached to only stop tracking it.", Safer: "git rm --cached <file>"})
		}
	case "filter-branch", "filter-repo":
		w = append(w, Warning{Risk: RiskHigh, Title: "Rewrites the entire history", Explanation: "Every commit hash changes. Everyone has to re-clone.", Backup: backupBranch()})
	case "gc", "prune", "reflog":
		if hasFlag(rest, "--prune=now", "--expire=now") || (sub == "reflog" && contains(rest, "expire")) {
			w = append(w, Warning{Risk: RiskHigh, Title: "Destroys your safety net", Explanation: "Unreachable commits and reflog entries are deleted; 'lost' work can no longer be recovered."})
		}
	}

	// most severe first
	for i := 1; i < len(w); i++ {
		for j := i; j > 0 && w[j].Risk > w[j-1].Risk; j-- {
			w[j], w[j-1] = w[j-1], w[j]
		}
	}
	return w
}

// MaxRisk returns the highest risk among warnings.
func MaxRisk(ws []Warning) Risk {
	r := RiskNone
	for _, w := range ws {
		r = max(r, w.Risk)
	}
	return r
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
