// Package advisor turns a repository status into next-step suggestions.
package advisor

import (
	"fmt"

	"gitwise/internal/git"
)

// Suggestion is a command the user may want to run next and why.
type Suggestion struct {
	Command string
	Reason  string
}

// Suggest returns the most useful next steps for the given status, most important first.
func Suggest(s *git.Status) []Suggestion {
	var out []Suggestion
	add := func(cmd, reason string) { out = append(out, Suggestion{cmd, reason}) }

	// An operation in progress always comes first: it blocks everything else.
	switch s.Operation {
	case "merge", "rebase", "cherry-pick", "revert":
		op := s.Operation
		cont := "git " + op + " --continue"
		if op == "merge" {
			cont = "git commit --no-edit"
		}
		if len(s.Conflicted) > 0 {
			add("<edit the conflicted files>", "remove the <<<<<<< ======= >>>>>>> markers and keep the code you want")
			add("git add <file>", "mark each file as resolved")
			add(cont, "then continue the "+op)
		} else {
			add(cont, "no conflicts left — finish the "+op)
		}
		add("git "+op+" --abort", "give up and go back to how things were before the "+op)
		return out
	case "bisect":
		add("git bisect good | git bisect bad", "tell Git whether the current commit works")
		add("git bisect reset", "end the bisect session and return to your branch")
		return out
	}

	if s.Detached {
		add("git switch -c <new-branch>", "you are on no branch — keep any commits by creating one")
		add("git switch -", "or go back to the branch you came from")
	}

	if len(s.Unstaged) > 0 {
		add("git diff", "review your unstaged edits")
		reason := "stage what should go into the next commit"
		if len(s.Untracked) > 0 {
			reason += fmt.Sprintf(" (also works for the %s)", plural(len(s.Untracked), "untracked file"))
		}
		add("git add <file>  (or git add -p)", reason)
	}
	if len(s.Untracked) > 0 && len(s.Unstaged) == 0 {
		add("git add <file>", fmt.Sprintf("start tracking %s — or add it to .gitignore", plural(len(s.Untracked), "new file")))
	}
	if len(s.Staged) > 0 {
		add("git diff --staged", "double-check what you are about to commit")
		add(`git commit -m "<message>"  (or gitwise commit)`, fmt.Sprintf("save %s as a commit", plural(len(s.Staged), "staged change")))
	}

	if s.Initial {
		if s.Clean() {
			add("git add <file> && git commit", "this repository has no commits yet — create your first one")
		}
		return out
	}

	switch {
	case s.Detached:
	case s.Upstream == "":
		add("git push -u origin "+s.Branch, "this branch isn't on the remote yet — publish it")
	case s.Ahead > 0 && s.Behind > 0:
		add("git pull --rebase  (or git pull)", fmt.Sprintf("your branch and %s diverged (%d ahead, %d behind) — integrate first", s.Upstream, s.Ahead, s.Behind))
	case s.Behind > 0:
		add("git pull", fmt.Sprintf("%s has %s you don't have", s.Upstream, plural(s.Behind, "new commit")))
	case s.Ahead > 0:
		add("git push", fmt.Sprintf("share your %s", plural(s.Ahead, "local commit")))
	}

	if s.Stashes > 0 {
		add("git stash list", fmt.Sprintf("you have %s — don't forget about them", plural(s.Stashes, "stash")))
	}

	if len(out) == 0 {
		add("git fetch", "everything is clean — check if teammates pushed something")
		add("git switch -c <feature-branch>", "start new work on its own branch")
	}
	return out
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}
