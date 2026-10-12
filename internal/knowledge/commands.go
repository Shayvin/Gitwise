// Package knowledge holds gitwise's built-in explanations of git.
package knowledge

import (
	"sort"
	"strings"
)

// Command describes a git subcommand in beginner-friendly terms.
type Command struct {
	Name     string
	Category string
	Summary  string
	Details  string
	Flags    []Flag
	Examples []Example
	Related  []string
}

// Flag is one option of a command.
type Flag struct {
	Names      []string // e.g. "-m", "--message"
	Desc       string
	TakesValue bool // the next argument belongs to this flag
}

// Example is a ready-to-use invocation.
type Example struct {
	Cmd  string
	Desc string
}

// Categories in display order.
var Categories = []string{"Setup", "Basics", "Branches", "History", "Remotes", "Undo", "Advanced"}

// Commands is the built-in command reference.
var Commands = []Command{
	{
		Name: "init", Category: "Setup",
		Summary: "Create a new, empty repository in the current folder.",
		Details: "Adds a hidden .git folder where Git stores all history. Your files are not touched and nothing is tracked until you `git add` it.",
		Flags: []Flag{
			{Names: []string{"-b", "--initial-branch"}, Desc: "name of the first branch (e.g. main)", TakesValue: true},
			{Names: []string{"--bare"}, Desc: "create a repository without a working folder (used for servers)"},
		},
		Examples: []Example{{"git init", "start tracking the current folder"}, {"git init -b main", "start with a branch called main"}},
		Related:  []string{"clone", "add"},
	},
	{
		Name: "clone", Category: "Setup",
		Summary: "Download a copy of an existing repository.",
		Details: "Copies every commit and branch, checks out the default branch and sets up a remote called `origin` pointing at the source.",
		Flags: []Flag{
			{Names: []string{"--depth"}, Desc: "only fetch the most recent N commits (faster, partial history)", TakesValue: true},
			{Names: []string{"-b", "--branch"}, Desc: "check out this branch instead of the default", TakesValue: true},
		},
		Examples: []Example{{"git clone <url>", "copy a repository into a new folder"}, {"git clone <url> my-dir", "copy into a folder called my-dir"}},
		Related:  []string{"remote", "fetch"},
	},
	{
		Name: "config", Category: "Setup",
		Summary: "Read or change Git settings.",
		Details: "Settings live at three levels: --system (whole machine), --global (your user) and --local (this repository, the default). The most specific one wins.",
		Flags: []Flag{
			{Names: []string{"--global"}, Desc: "apply to every repository of your user"},
			{Names: []string{"--local"}, Desc: "apply only to this repository"},
			{Names: []string{"--list", "-l"}, Desc: "show all settings"},
			{Names: []string{"--unset"}, Desc: "remove a setting"},
		},
		Examples: []Example{
			{`git config --global user.name "Your Name"`, "set the name recorded in your commits"},
			{`git config --global user.email "you@example.com"`, "set the email recorded in your commits"},
			{"git config --list --show-origin", "see every setting and which file it comes from"},
		},
	},
	{
		Name: "status", Category: "Basics",
		Summary: "Show what changed since the last commit.",
		Details: "Lists staged changes (will go into the next commit), unstaged changes (edited but not added) and untracked files (Git has never seen them). Run it often — it is always safe.",
		Flags: []Flag{
			{Names: []string{"-s", "--short"}, Desc: "compact two-column output"},
			{Names: []string{"-b", "--branch"}, Desc: "include branch and tracking info"},
		},
		Examples: []Example{{"git status", "what is going on?"}, {"git status -sb", "short version with branch info"}},
		Related:  []string{"diff", "add"},
	},
	{
		Name: "add", Category: "Basics",
		Summary: "Stage changes so they are included in the next commit.",
		Details: "The staging area (index) is a draft of your next commit. `add` copies the current version of a file into it. Editing the file again afterwards requires another `add`.",
		Flags: []Flag{
			{Names: []string{"-A", "--all"}, Desc: "stage every change, including deletions and new files"},
			{Names: []string{"-p", "--patch"}, Desc: "interactively choose which pieces (hunks) to stage"},
			{Names: []string{"-u", "--update"}, Desc: "stage changes to already-tracked files only"},
			{Names: []string{"-n", "--dry-run"}, Desc: "show what would be staged without doing it"},
		},
		Examples: []Example{
			{"git add <file>", "stage one file"},
			{"git add .", "stage everything in the current folder"},
			{"git add -p", "review and stage changes piece by piece"},
		},
		Related: []string{"restore", "commit", "status"},
	},
	{
		Name: "commit", Category: "Basics",
		Summary: "Save the staged changes as a new snapshot in history.",
		Details: "A commit records the staged files, a message, the author and a pointer to its parent commit. Good messages start with a short imperative summary like \"Add login form\".",
		Flags: []Flag{
			{Names: []string{"-m", "--message"}, Desc: "commit message (without it Git opens your editor)", TakesValue: true},
			{Names: []string{"-a", "--all"}, Desc: "automatically stage modified tracked files first (not new files)"},
			{Names: []string{"--amend"}, Desc: "replace the last commit instead of adding a new one — rewrites history"},
			{Names: []string{"--no-edit"}, Desc: "keep the existing/default message"},
			{Names: []string{"-v", "--verbose"}, Desc: "show the diff in the editor while writing the message"},
		},
		Examples: []Example{
			{`git commit -m "Add login form"`, "commit staged changes with a message"},
			{"git commit --amend", "fix the last commit (message or content)"},
		},
		Related: []string{"add", "log", "reset"},
	},
	{
		Name: "diff", Category: "Basics",
		Summary: "Show line-by-line changes.",
		Details: "Without arguments it shows unstaged changes. Lines starting with + were added, lines with - were removed.",
		Flags: []Flag{
			{Names: []string{"--staged", "--cached"}, Desc: "show what is staged (what the next commit will contain)"},
			{Names: []string{"--stat"}, Desc: "summary of changed files and line counts"},
			{Names: []string{"--word-diff"}, Desc: "highlight changed words instead of whole lines"},
		},
		Examples: []Example{
			{"git diff", "what did I change but not stage?"},
			{"git diff --staged", "what am I about to commit?"},
			{"git diff main..feature", "compare two branches"},
		},
		Related: []string{"status", "log", "show"},
	},
	{
		Name: "rm", Category: "Basics",
		Summary: "Delete a file and stage the deletion.",
		Flags: []Flag{
			{Names: []string{"--cached"}, Desc: "stop tracking the file but keep it on disk"},
			{Names: []string{"-r"}, Desc: "remove folders recursively"},
			{Names: []string{"-f", "--force"}, Desc: "remove even if the file has unsaved changes"},
		},
		Examples: []Example{{"git rm <file>", "delete and stage the deletion"}, {"git rm --cached <file>", "untrack a file (e.g. after adding it to .gitignore)"}},
	},
	{
		Name: "mv", Category: "Basics",
		Summary:  "Move or rename a file and stage the change.",
		Examples: []Example{{"git mv old.txt new.txt", "rename a tracked file"}},
	},
	{
		Name: "branch", Category: "Branches",
		Summary: "List, create, rename or delete branches.",
		Details: "A branch is just a movable label pointing at a commit. Creating one is cheap and does not switch to it (use `git switch -c` to create and switch).",
		Flags: []Flag{
			{Names: []string{"-a", "--all"}, Desc: "include remote-tracking branches"},
			{Names: []string{"-v", "-vv"}, Desc: "show last commit (and upstream with -vv)"},
			{Names: []string{"-d", "--delete"}, Desc: "delete a branch that is fully merged"},
			{Names: []string{"-D"}, Desc: "force-delete a branch, even with unmerged commits"},
			{Names: []string{"-m", "--move"}, Desc: "rename a branch"},
			{Names: []string{"--merged"}, Desc: "list branches already merged into the current one"},
		},
		Examples: []Example{
			{"git branch", "list local branches"},
			{"git branch -vv", "branches with their upstream and ahead/behind"},
			{"git branch -d <name>", "delete a merged branch"},
			{"git branch -m <new-name>", "rename the current branch"},
		},
		Related: []string{"switch", "merge"},
	},
	{
		Name: "switch", Category: "Branches",
		Summary: "Change to another branch.",
		Details: "The modern, safer replacement for switching with `git checkout`. Uncommitted changes come along with you if they do not conflict.",
		Flags: []Flag{
			{Names: []string{"-c", "--create"}, Desc: "create a new branch and switch to it", TakesValue: true},
			{Names: []string{"-"}, Desc: "switch back to the previous branch"},
			{Names: []string{"--detach"}, Desc: "check out a commit without a branch (detached HEAD)"},
		},
		Examples: []Example{
			{"git switch main", "go to the main branch"},
			{"git switch -c feature/login", "create a branch and switch to it"},
			{"git switch -", "go back to the previous branch"},
		},
		Related: []string{"branch", "checkout"},
	},
	{
		Name: "checkout", Category: "Branches",
		Summary: "Old all-in-one command: switch branches OR restore files.",
		Details: "Because it does two very different things, Git added `switch` (branches) and `restore` (files). `git checkout -- <file>` silently throws away your edits.",
		Flags: []Flag{
			{Names: []string{"-b"}, Desc: "create a new branch and switch to it", TakesValue: true},
			{Names: []string{"--"}, Desc: "everything after this is a file path, not a branch"},
		},
		Examples: []Example{{"git checkout <branch>", "switch branch (prefer git switch)"}, {"git checkout -b <branch>", "create + switch (prefer git switch -c)"}},
		Related:  []string{"switch", "restore"},
	},
	{
		Name: "merge", Category: "Branches",
		Summary: "Combine another branch into the current one.",
		Details: "If the current branch has not moved, Git just moves the label forward (fast-forward). Otherwise it creates a merge commit with two parents. Conflicts happen when both sides changed the same lines.",
		Flags: []Flag{
			{Names: []string{"--no-ff"}, Desc: "always create a merge commit, even when fast-forward is possible"},
			{Names: []string{"--ff-only"}, Desc: "refuse unless a fast-forward is possible"},
			{Names: []string{"--squash"}, Desc: "combine all changes into one staged change, without a merge commit"},
			{Names: []string{"--abort"}, Desc: "cancel a merge with conflicts and go back to before it started"},
			{Names: []string{"--continue"}, Desc: "finish the merge after resolving conflicts"},
		},
		Examples: []Example{{"git merge <branch>", "bring <branch> into the current branch"}, {"git merge --abort", "give up on a conflicted merge"}},
		Related:  []string{"rebase", "branch"},
	},
	{
		Name: "rebase", Category: "Advanced",
		Summary: "Replay your commits on top of another base.",
		Details: "Produces a straight line of history by creating NEW copies of your commits. Never rebase commits other people already pulled — their history would no longer match yours.",
		Flags: []Flag{
			{Names: []string{"-i", "--interactive"}, Desc: "edit, reorder, squash or drop commits in an editor"},
			{Names: []string{"--continue"}, Desc: "continue after resolving a conflict"},
			{Names: []string{"--abort"}, Desc: "stop and return to the state before the rebase"},
			{Names: []string{"--skip"}, Desc: "drop the commit that is currently conflicting"},
			{Names: []string{"--onto"}, Desc: "rebase onto a different base than the upstream", TakesValue: true},
		},
		Examples: []Example{
			{"git rebase main", "move your branch's commits on top of main"},
			{"git rebase -i HEAD~3", "clean up the last 3 commits"},
			{"git rebase --abort", "get out of a rebase that went wrong"},
		},
		Related: []string{"merge", "reflog"},
	},
	{
		Name: "log", Category: "History",
		Summary: "Show the commit history.",
		Flags: []Flag{
			{Names: []string{"--oneline"}, Desc: "one short line per commit"},
			{Names: []string{"--graph"}, Desc: "draw branch and merge lines"},
			{Names: []string{"--all"}, Desc: "include every branch, not just the current one"},
			{Names: []string{"--decorate"}, Desc: "show branch and tag names next to commits"},
			{Names: []string{"-p", "--patch"}, Desc: "show the diff of each commit"},
			{Names: []string{"-n"}, Desc: "limit to N commits", TakesValue: true},
			{Names: []string{"--author"}, Desc: "only commits by this author", TakesValue: true},
			{Names: []string{"--since"}, Desc: "only commits newer than a date, e.g. \"2 weeks ago\"", TakesValue: true},
			{Names: []string{"-S"}, Desc: "find commits that added or removed this text", TakesValue: true},
		},
		Examples: []Example{
			{"git log --oneline --graph --all", "visual overview of all branches"},
			{"git log -p <file>", "full history of one file"},
			{`git log -S "functionName"`, "find when some code appeared or disappeared"},
		},
		Related: []string{"show", "diff", "reflog"},
	},
	{
		Name: "show", Category: "History",
		Summary:  "Show one commit (message and diff) or any other object.",
		Examples: []Example{{"git show", "inspect the latest commit"}, {"git show <commit>:<file>", "see a file as it was in a commit"}},
	},
	{
		Name: "blame", Category: "History",
		Summary:  "Show who last changed each line of a file, and in which commit.",
		Flags:    []Flag{{Names: []string{"-L"}, Desc: "only a range of lines, e.g. -L 10,20", TakesValue: true}},
		Examples: []Example{{"git blame <file>", "line-by-line authorship"}},
	},
	{
		Name: "tag", Category: "History",
		Summary: "Give a permanent name to a commit, usually a release.",
		Flags: []Flag{
			{Names: []string{"-a"}, Desc: "create an annotated tag (with author, date and message)"},
			{Names: []string{"-m"}, Desc: "tag message", TakesValue: true},
			{Names: []string{"-d"}, Desc: "delete a tag"},
		},
		Examples: []Example{{`git tag -a v1.0 -m "First release"`, "tag the current commit"}, {"git push --tags", "upload tags to the remote"}},
	},
	{
		Name: "remote", Category: "Remotes",
		Summary: "Manage the list of servers your repository talks to.",
		Flags:   []Flag{{Names: []string{"-v"}, Desc: "show URLs"}},
		Examples: []Example{
			{"git remote -v", "list remotes and their URLs"},
			{"git remote add origin <url>", "connect a local repo to a server"},
			{"git remote set-url origin <url>", "change a remote's URL"},
		},
		Related: []string{"fetch", "push", "pull"},
	},
	{
		Name: "fetch", Category: "Remotes",
		Summary: "Download new commits from a remote without changing your files.",
		Details: "Updates remote-tracking branches like origin/main. Always safe — it never touches your work. Combine with `git status` to see if you are behind.",
		Flags: []Flag{
			{Names: []string{"--all"}, Desc: "fetch from every remote"},
			{Names: []string{"-p", "--prune"}, Desc: "forget remote branches that were deleted on the server"},
		},
		Examples: []Example{{"git fetch", "check for new work on the server"}, {"git fetch --prune", "also clean up deleted remote branches"}},
		Related:  []string{"pull", "merge"},
	},
	{
		Name: "pull", Category: "Remotes",
		Summary: "Fetch from the remote and integrate it into your branch.",
		Details: "Equivalent to `git fetch` followed by `git merge` (or `git rebase` with --rebase). Commit or stash your work first.",
		Flags: []Flag{
			{Names: []string{"--rebase"}, Desc: "rebase your local commits on top instead of merging"},
			{Names: []string{"--ff-only"}, Desc: "only update if no merge is needed"},
		},
		Examples: []Example{{"git pull", "get the latest changes"}, {"git pull --rebase", "get changes and keep history linear"}},
		Related:  []string{"fetch", "push"},
	},
	{
		Name: "push", Category: "Remotes",
		Summary: "Upload your commits to a remote.",
		Details: "Git refuses if the remote has commits you don't have — pull first. The first push of a new branch needs -u to link it to the remote branch.",
		Flags: []Flag{
			{Names: []string{"-u", "--set-upstream"}, Desc: "remember the remote branch so future `git push`/`git pull` need no arguments"},
			{Names: []string{"-f", "--force"}, Desc: "overwrite the remote branch — can destroy other people's work"},
			{Names: []string{"--force-with-lease"}, Desc: "force push, but only if nobody else pushed in the meantime"},
			{Names: []string{"-d", "--delete"}, Desc: "delete a branch on the remote"},
			{Names: []string{"--tags"}, Desc: "push all tags"},
		},
		Examples: []Example{
			{"git push", "upload commits on the current branch"},
			{"git push -u origin <branch>", "first push of a new branch"},
			{"git push --force-with-lease", "safely push after rewriting your own branch"},
		},
		Related: []string{"pull", "fetch"},
	},
	{
		Name: "restore", Category: "Undo",
		Summary: "Discard changes in files or unstage them.",
		Details: "Without --staged it overwrites your file with the last committed version — unsaved edits are lost for good. With --staged it only removes the file from the staging area.",
		Flags: []Flag{
			{Names: []string{"--staged", "-S"}, Desc: "unstage (keep your edits in the file)"},
			{Names: []string{"--source", "-s"}, Desc: "restore from a specific commit", TakesValue: true},
			{Names: []string{"-p", "--patch"}, Desc: "choose hunks interactively"},
		},
		Examples: []Example{
			{"git restore --staged <file>", "unstage a file"},
			{"git restore <file>", "throw away edits to a file"},
			{"git restore --source HEAD~1 <file>", "bring back the previous version of a file"},
		},
		Related: []string{"reset", "add"},
	},
	{
		Name: "reset", Category: "Undo",
		Summary: "Move the current branch to another commit.",
		Details: "--soft keeps changes staged, --mixed (default) keeps them unstaged, --hard throws them away. Commits you reset away can be found again in `git reflog` for a while.",
		Flags: []Flag{
			{Names: []string{"--soft"}, Desc: "move the branch, keep changes staged"},
			{Names: []string{"--mixed"}, Desc: "move the branch, keep changes as unstaged edits (default)"},
			{Names: []string{"--hard"}, Desc: "move the branch AND discard all uncommitted changes"},
		},
		Examples: []Example{
			{"git reset --soft HEAD~1", "undo the last commit, keep its changes staged"},
			{"git reset <file>", "unstage a file (same as restore --staged)"},
			{"git reset --hard origin/main", "make your branch exactly match the remote (destructive)"},
		},
		Related: []string{"restore", "revert", "reflog"},
	},
	{
		Name: "revert", Category: "Undo",
		Summary: "Create a new commit that undoes an earlier commit.",
		Details: "The safe way to undo something that was already pushed: history is not rewritten, the undo is just another commit.",
		Flags: []Flag{
			{Names: []string{"--no-edit"}, Desc: "use the default message"},
			{Names: []string{"-n", "--no-commit"}, Desc: "apply the undo but don't commit yet"},
		},
		Examples: []Example{{"git revert <commit>", "undo a commit by adding its opposite"}, {"git revert HEAD", "undo the latest commit"}},
		Related:  []string{"reset"},
	},
	{
		Name: "stash", Category: "Undo",
		Summary: "Put uncommitted changes aside temporarily.",
		Details: "Stashing cleans your working folder so you can switch branches or pull, then brings the changes back later.",
		Flags: []Flag{
			{Names: []string{"-u", "--include-untracked"}, Desc: "also stash new, untracked files"},
			{Names: []string{"-m", "--message"}, Desc: "describe the stash", TakesValue: true},
		},
		Examples: []Example{
			{"git stash", "save changes aside"},
			{"git stash list", "show saved stashes"},
			{"git stash pop", "re-apply the latest stash and remove it"},
			{"git stash apply stash@{1}", "re-apply a specific stash but keep it"},
		},
		Related: []string{"switch", "pull"},
	},
	{
		Name: "reflog", Category: "Undo",
		Summary:  "Diary of everywhere HEAD has been — your safety net.",
		Details:  "Every commit, reset, rebase and checkout is recorded locally for ~90 days. If you 'lost' a commit, it is almost certainly in here.",
		Examples: []Example{{"git reflog", "see recent HEAD positions"}, {"git switch -c rescue HEAD@{2}", "recover the state from two moves ago"}},
		Related:  []string{"reset", "log"},
	},
	{
		Name: "clean", Category: "Undo",
		Summary: "Permanently delete untracked files.",
		Details: "Deleted files do NOT go to the trash and Git cannot bring them back. Always run with -n first.",
		Flags: []Flag{
			{Names: []string{"-n", "--dry-run"}, Desc: "only show what would be deleted"},
			{Names: []string{"-f", "--force"}, Desc: "actually delete"},
			{Names: []string{"-d"}, Desc: "also delete untracked folders"},
			{Names: []string{"-x"}, Desc: "also delete ignored files (build output, .env, ...)"},
			{Names: []string{"-i", "--interactive"}, Desc: "ask for each file"},
		},
		Examples: []Example{{"git clean -n", "preview what would be deleted"}, {"git clean -fd", "delete untracked files and folders"}},
	},
	{
		Name: "cherry-pick", Category: "Advanced",
		Summary: "Copy a single commit from somewhere else onto the current branch.",
		Flags: []Flag{
			{Names: []string{"--continue"}, Desc: "continue after resolving conflicts"},
			{Names: []string{"--abort"}, Desc: "cancel and go back"},
		},
		Examples: []Example{{"git cherry-pick <commit>", "apply that commit here"}},
	},
	{
		Name: "bisect", Category: "Advanced",
		Summary: "Binary-search history to find the commit that introduced a bug.",
		Examples: []Example{
			{"git bisect start", "begin a search"},
			{"git bisect bad", "mark the current commit as broken"},
			{"git bisect good <commit>", "mark a known-good commit"},
			{"git bisect reset", "finish and return to where you started"},
		},
	},
}

// Lookup finds a command by name.
func Lookup(name string) (*Command, bool) {
	for i := range Commands {
		if Commands[i].Name == name {
			return &Commands[i], true
		}
	}
	return nil, false
}

// FindFlag returns the flag matching arg (handles --flag=value).
func (c *Command) FindFlag(arg string) *Flag {
	name, _, _ := strings.Cut(arg, "=")
	for i := range c.Flags {
		for _, n := range c.Flags[i].Names {
			if n == name {
				return &c.Flags[i]
			}
		}
	}
	return nil
}

// Names returns all known command names, sorted.
func Names() []string {
	names := make([]string, len(Commands))
	for i, c := range Commands {
		names[i] = c.Name
	}
	sort.Strings(names)
	return names
}

// Closest returns the known command name nearest to s by edit distance,
// or "" if nothing is reasonably close.
func Closest(s string) string {
	best, bestDist := "", 3
	for _, c := range Commands {
		if d := levenshtein(s, c.Name); d < bestDist {
			best, bestDist = c.Name, d
		}
	}
	return best
}

func levenshtein(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
