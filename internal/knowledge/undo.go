package knowledge

import (
	"strings"

	"gitwise/internal/git"
)

// UndoRecipe solves one "oh no, how do I undo this?" situation.
type UndoRecipe struct {
	Title    string
	Problem  string
	Steps    []string // git commands; <placeholders> must be filled in by the user
	Explain  string
	Risk     Risk
	Relevant func(*git.Status) bool // highlights the recipe for the current state
}

// Runnable reports whether every step can run without user input.
func (r UndoRecipe) Runnable() bool {
	for _, s := range r.Steps {
		if strings.Contains(s, "<") {
			return false
		}
	}
	return len(r.Steps) > 0
}

// UndoRecipes is the list of common undo situations.
var UndoRecipes = []UndoRecipe{
	{
		Title:    "Abort the merge in progress",
		Problem:  "A merge produced conflicts and I want to go back to how things were.",
		Steps:    []string{"git merge --abort"},
		Explain:  "Restores your branch and files to the state before `git merge` started.",
		Risk:     RiskLow,
		Relevant: func(s *git.Status) bool { return s.Operation == "merge" },
	},
	{
		Title:    "Abort the rebase in progress",
		Problem:  "A rebase is stuck or went wrong.",
		Steps:    []string{"git rebase --abort"},
		Explain:  "Puts your branch back exactly where it was before the rebase started.",
		Risk:     RiskLow,
		Relevant: func(s *git.Status) bool { return s.Operation == "rebase" },
	},
	{
		Title:    "Abort the cherry-pick in progress",
		Problem:  "A cherry-pick hit a conflict and I want to cancel it.",
		Steps:    []string{"git cherry-pick --abort"},
		Risk:     RiskLow,
		Relevant: func(s *git.Status) bool { return s.Operation == "cherry-pick" },
	},
	{
		Title:    "Unstage everything",
		Problem:  "I ran `git add` on files I don't want in the next commit.",
		Steps:    []string{"git restore --staged ."},
		Explain:  "Removes the files from the staging area. Your edits stay in the files.",
		Risk:     RiskNone,
		Relevant: func(s *git.Status) bool { return len(s.Staged) > 0 },
	},
	{
		Title:   "Unstage one file",
		Problem: "One file was staged by accident.",
		Steps:   []string{"git restore --staged <file>"},
		Explain: "The file keeps your edits; it just won't be part of the next commit.",
		Risk:    RiskNone,
	},
	{
		Title:   "Discard my edits to a file",
		Problem: "I messed up a file and want the last committed version back.",
		Steps:   []string{"git restore <file>"},
		Explain: "Overwrites the file with the version from the last commit. Your edits are gone for good — `git stash` instead if unsure.",
		Risk:    RiskHigh,
	},
	{
		Title:    "Put my changes aside (stash)",
		Problem:  "I want a clean working tree but keep my work for later.",
		Steps:    []string{"git stash push -u"},
		Explain:  "Saves all changes including new files. Get them back with `git stash pop`.",
		Risk:     RiskNone,
		Relevant: func(s *git.Status) bool { return !s.Clean() && s.Operation == "" },
	},
	{
		Title:    "Change the last commit message",
		Problem:  "I made a typo in my last commit message.",
		Steps:    []string{`git commit --amend -m "<new message>"`},
		Explain:  "Replaces the last commit. Only do this if you haven't pushed it yet.",
		Risk:     RiskLow,
		Relevant: unpushed,
	},
	{
		Title:    "Add a forgotten file to the last commit",
		Problem:  "I committed but forgot to include a file.",
		Steps:    []string{"git add <file>", "git commit --amend --no-edit"},
		Explain:  "Adds the file into the previous commit without changing its message. Only for unpushed commits.",
		Risk:     RiskLow,
		Relevant: func(s *git.Status) bool { return !s.Initial },
	},
	{
		Title:    "Undo the last commit, keep the changes",
		Problem:  "I committed too early and want to keep working on it.",
		Steps:    []string{"git reset --soft HEAD~1"},
		Explain:  "The commit disappears but all its changes are staged again, ready to be re-committed.",
		Risk:     RiskLow,
		Relevant: unpushed,
	},
	{
		Title:   "Undo a commit that is already pushed",
		Problem: "A bad commit is on the server and others may have it.",
		Steps:   []string{"git revert <commit>", "git push"},
		Explain: "Adds a new commit that reverses the bad one. History stays intact so nobody's copy breaks.",
		Risk:    RiskNone,
	},
	{
		Title:   "I committed to the wrong branch",
		Problem: "My last commit should have gone on a new branch, not this one.",
		Steps:   []string{"git branch <new-branch>", "git reset --hard HEAD~1", "git switch <new-branch>"},
		Explain: "Creates a branch that keeps the commit, moves the current branch back one commit, then switches to the new branch. Commit or stash other changes first.",
		Risk:    RiskMedium,
	},
	{
		Title:   "Recover a lost commit or deleted branch",
		Problem: "After a reset, rebase or branch -D my work seems gone.",
		Steps:   []string{"git reflog", "git switch -c rescue <hash-from-reflog>"},
		Explain: "The reflog lists every commit HEAD pointed to. Find the one you want and create a branch on it.",
		Risk:    RiskNone,
	},
	{
		Title:    "Leave detached HEAD",
		Problem:  "Git says 'You are in detached HEAD state'.",
		Steps:    []string{"git switch -c <new-branch>"},
		Explain:  "Saves any commits you made into a new branch. If you made none, `git switch -` simply takes you back.",
		Risk:     RiskNone,
		Relevant: func(s *git.Status) bool { return s.Detached },
	},
	{
		Title:   "Make my branch match the remote exactly",
		Problem: "My local branch is a mess, I want what's on the server.",
		Steps:   []string{"git fetch", "git reset --hard @{u}"},
		Explain: "Throws away all local commits and uncommitted changes on this branch. Consider a backup branch first.",
		Risk:    RiskHigh,
	},
	{
		Title:   "Stop tracking a file but keep it",
		Problem: "I committed a file that should be in .gitignore (e.g. .env).",
		Steps:   []string{"git rm --cached <file>", `git commit -m "Stop tracking <file>"`},
		Explain: "Removes the file from the repository but leaves it on disk. Add it to .gitignore too. If it contains secrets, rotate them — they're still in history.",
		Risk:    RiskLow,
	},
}

// unpushed reports whether the last commit probably exists only locally.
func unpushed(s *git.Status) bool {
	return !s.Initial && (s.Upstream == "" || s.Ahead > 0)
}
