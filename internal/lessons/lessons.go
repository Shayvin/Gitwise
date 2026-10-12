package lessons

import (
	"regexp"
	"strings"
)

// Attempt is what the learner just did, passed to step checks.
type Attempt struct {
	Line   string // the command line they typed
	Output string
}

// ran reports whether the command line invokes the given git subcommand.
func (a Attempt) ran(sub string) bool {
	return regexp.MustCompile(`(^|[;&|]\s*)git\s+(\S+\s+)*?` + regexp.QuoteMeta(sub) + `\b`).MatchString(a.Line)
}

// Step is one task inside a lesson.
type Step struct {
	Goal    string // short headline
	Text    string // explanation; `backticks` render as code
	Hint    string // the exact command(s) to type
	Prepare func(*Sandbox) error
	Done    func(*Sandbox, Attempt) bool
}

// Lesson is a sequence of steps sharing one sandbox.
type Lesson struct {
	ID      string
	Title   string
	Summary string
	Setup   func(*Sandbox) error
	Steps   []Step
}

// All lessons in recommended order.
var All = []Lesson{
	{
		ID: "first-commit", Title: "1. Your first commit",
		Summary: "init, status, add, commit, log — the everyday loop",
		Steps: []Step{
			{
				Goal: "Create a repository",
				Text: "This folder is empty and not tracked yet. Turn it into a Git repository with `git init`. Git creates a hidden .git folder that stores all history.",
				Hint: "git init",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.exists(".git") },
			},
			{
				Goal: "Create a file and check the status",
				Text: "Create a README with `echo \"# My Project\" > README.md` and then ask Git what it sees with `git status`. The file shows up as *untracked*: Git notices it but doesn't track it yet.",
				Hint: `echo "# My Project" > README.md; git status`,
				Done: func(sb *Sandbox, a Attempt) bool { return sb.exists("README.md") && a.ran("status") },
			},
			{
				Goal: "Stage the file",
				Text: "The staging area is a draft of your next commit. Put README.md into it with `git add README.md`, then run `git status` again if you like — it is now listed under 'Changes to be committed'.",
				Hint: "git add README.md",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.isStaged("README.md") || sb.commitCount("HEAD") > 0 },
			},
			{
				Goal: "Commit",
				Text: "Save the snapshot with a message describing *what* and *why*: `git commit -m \"Add README\"`. Messages are written in the imperative: \"Add\", not \"Added\".",
				Hint: `git commit -m "Add README"`,
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.commitCount("HEAD") >= 1 },
			},
			{
				Goal: "Look at the history",
				Text: "Every commit has a unique hash. See your history with `git log --oneline`.",
				Hint: "git log --oneline",
				Done: func(_ *Sandbox, a Attempt) bool { return a.ran("log") },
			},
		},
	},
	{
		ID: "branches", Title: "2. Branching and merging",
		Summary: "work on features in isolation, then bring them together",
		Setup: func(sb *Sandbox) error {
			return sb.script("project", "git init -q", `echo "# Shop" > README.md`, "git add .", `git commit -qm "Initial commit"`)
		},
		Steps: []Step{
			{
				Goal: "Create a feature branch",
				Text: "Branches let you work without disturbing `main`. Create a new branch and switch to it in one go with `git switch -c feature/greeting`.",
				Hint: "git switch -c feature/greeting",
				Done: func(sb *Sandbox, _ Attempt) bool { b := sb.branch(); return b != "" && b != "main" },
			},
			{
				Goal: "Commit on the branch",
				Text: "Create a file and commit it on your branch: `echo hello > hello.txt`, then `git add hello.txt` and `git commit -m \"Add greeting\"`. You can chain commands with `&&`.",
				Hint: `echo hello > hello.txt && git add hello.txt && git commit -m "Add greeting"`,
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.branch() != "main" && sb.commitCount("main..HEAD") >= 1 },
			},
			{
				Goal: "Switch back to main",
				Text: "Go back with `git switch main`. Run `ls` afterwards: hello.txt is gone — it only exists on the feature branch!",
				Hint: "git switch main && ls",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.branch() == "main" },
			},
			{
				Goal: "Merge the feature",
				Text: "Bring the feature into main with `git merge feature/greeting`. Since main didn't move in the meantime, Git simply *fast-forwards* the main label.",
				Hint: "git merge feature/greeting",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.branch() == "main" && sb.commitCount("HEAD") >= 2 },
			},
			{
				Goal: "Clean up",
				Text: "The branch is merged, so it can go: `git branch -d feature/greeting`. `-d` refuses to delete unmerged work, which makes it safe.",
				Hint: "git branch -d feature/greeting",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.Git("branch", "--format=%(refname:short)") == "main" },
			},
		},
	},
	{
		ID: "undo", Title: "3. Undoing mistakes",
		Summary: "restore, unstage, amend, reset and the reflog safety net",
		Setup: func(sb *Sandbox) error {
			return sb.script("project",
				"git init -q",
				`echo "version 1" > app.txt`, "git add .", `git commit -qm "Add app"`,
				`echo "TODO" > notes.txt`, "git add .", `git commit -qm "Add notes"`,
				`echo "BROKEN ### do not ship" >> app.txt`)
		},
		Steps: []Step{
			{
				Goal: "Discard broken edits",
				Text: "Someone broke app.txt. Look at the damage with `git diff`, then throw the edits away with `git restore app.txt`. Careful: restored-away edits are gone for good.",
				Hint: "git diff; git restore app.txt",
				Done: func(sb *Sandbox, _ Attempt) bool { return !sb.isDirty("app.txt") },
			},
			{
				Goal: "Unstage a file",
				Prepare: func(sb *Sandbox) error {
					return sb.script("project", `echo "password=hunter2" > secret.txt`, "git add secret.txt")
				},
				Text: "Oops — secret.txt (a password!) was just staged. Check with `git status`, then take it out of the staging area with `git restore --staged secret.txt`. The file stays on disk.",
				Hint: "git restore --staged secret.txt",
				Done: func(sb *Sandbox, _ Attempt) bool { return !sb.isStaged("secret.txt") && sb.exists("secret.txt") },
			},
			{
				Goal: "Fix a commit message",
				Prepare: func(sb *Sandbox) error {
					return sb.script("project", "rm -f secret.txt", `echo "shiny" > feature.txt`, "git add feature.txt", `git commit -qm "Add featuer"`)
				},
				Text: "A commit with a typo was just made (run `git log --oneline`). Since it isn't pushed, you can replace it: `git commit --amend -m \"Add feature\"`.",
				Hint: `git commit --amend -m "Add feature"`,
				Done: func(sb *Sandbox, _ Attempt) bool {
					msg := sb.Git("log", "-1", "--format=%s")
					return strings.Contains(strings.ToLower(msg), "feature") && !strings.Contains(msg, "featuer")
				},
			},
			{
				Goal: "Undo a commit but keep its changes",
				Text: "Actually, that commit was premature. `git reset --soft HEAD~1` moves the branch back one commit (HEAD~1 = \"one before HEAD\") but keeps the changes staged.",
				Hint: "git reset --soft HEAD~1",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.commitCount("HEAD") == 2 && sb.isStaged("feature.txt") },
			},
			{
				Goal: "Meet the reflog",
				Text: "Even the commit you just 'deleted' isn't lost. `git reflog` lists every place HEAD has been — your ultimate undo button.",
				Hint: "git reflog",
				Done: func(_ *Sandbox, a Attempt) bool { return a.ran("reflog") },
			},
		},
	},
	{
		ID: "conflicts", Title: "4. Resolving a merge conflict",
		Summary: "what conflict markers mean and how to finish the merge",
		Setup: func(sb *Sandbox) error {
			return sb.script("project",
				"git init -q",
				`echo "The hero walked into the forest." > story.txt`, "git add .", `git commit -qm "Start story"`,
				"git switch -qc night",
				`echo "The hero walked into the dark forest at night." > story.txt`, `git commit -qam "Make it night"`,
				"git switch -q main",
				`echo "The hero walked into the sunny forest." > story.txt`, `git commit -qam "Make it sunny"`)
		},
		Steps: []Step{
			{
				Goal: "Start the merge",
				Text: "Both `main` and the `night` branch changed the same line of story.txt. Merge with `git merge night` and watch Git report a CONFLICT.",
				Hint: "git merge night",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.inMerge() },
			},
			{
				Goal: "Inspect the conflict",
				Text: "Run `cat story.txt`. Between `<<<<<<< HEAD` and `=======` is your version, between `=======` and `>>>>>>> night` is theirs. `git status` lists the file as 'both modified'.",
				Hint: "cat story.txt",
				Done: func(_ *Sandbox, a Attempt) bool {
					return strings.Contains(a.Line, "story.txt") || a.ran("status") || a.ran("diff")
				},
			},
			{
				Goal: "Resolve it",
				Text: "Normally you'd edit the file in your editor. Here, overwrite it with the combined version: `echo \"The hero walked into the dark, sunny forest.\" > story.txt`. Every marker line must be gone.",
				Hint: `echo "The hero walked into the dark, sunny forest." > story.txt`,
				Done: func(sb *Sandbox, _ Attempt) bool {
					c := sb.read("story.txt")
					return c != "" && !strings.Contains(c, "<<<<<<<") && !strings.Contains(c, ">>>>>>>")
				},
			},
			{
				Goal: "Mark as resolved",
				Text: "Tell Git the conflict is solved by staging the file: `git add story.txt`.",
				Hint: "git add story.txt",
				Done: func(sb *Sandbox, _ Attempt) bool {
					return sb.Git("diff", "--name-only", "--diff-filter=U") == "" && sb.isStaged("story.txt")
				},
			},
			{
				Goal: "Finish the merge",
				Text: "Conclude with `git commit --no-edit`, which uses Git's default merge message. Then admire the result with `git log --oneline --graph`.",
				Hint: "git commit --no-edit",
				Done: func(sb *Sandbox, _ Attempt) bool {
					return !sb.inMerge() && len(strings.Fields(sb.Git("rev-list", "--parents", "-n1", "HEAD"))) == 3
				},
			},
		},
	},
	{
		ID: "stash", Title: "5. Stashing work in progress",
		Summary: "park unfinished changes and bring them back later",
		Setup: func(sb *Sandbox) error {
			return sb.script("project",
				"git init -q", `echo "v1" > app.txt`, "git add .", `git commit -qm "Add app"`,
				`echo "half-finished idea" >> app.txt`)
		},
		Steps: []Step{
			{
				Goal: "Stash your changes",
				Text: "You're in the middle of something (see `git status`) but need a clean working tree for an urgent fix. Park the work with `git stash`.",
				Hint: "git stash",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.stashCount() >= 1 && !sb.isDirty("app.txt") },
			},
			{
				Goal: "List your stashes",
				Text: "The working tree is clean now. See what's parked with `git stash list`.",
				Hint: "git stash list",
				Done: func(_ *Sandbox, a Attempt) bool { return a.ran("stash") && strings.Contains(a.Line, "list") },
			},
			{
				Goal: "Bring it back",
				Text: "Urgent fix done — continue where you left off with `git stash pop`. It re-applies the stash and removes it from the list.",
				Hint: "git stash pop",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.stashCount() == 0 && sb.isDirty("app.txt") },
			},
		},
	},
	{
		ID: "remotes", Title: "6. Working with a remote",
		Summary: "push, fetch and pull with a (simulated) teammate",
		Setup: func(sb *Sandbox) error {
			return sb.script("project",
				"git init -q --bare ../origin.git",
				"git init -q", `echo "# Team project" > README.md`, "git add .", `git commit -qm "Initial commit"`)
		},
		Steps: []Step{
			{
				Goal: "Connect a remote",
				Text: "A 'server' repository was created at ../origin.git. Register it under the conventional name origin: `git remote add origin ../origin.git`. Verify with `git remote -v`.",
				Hint: "git remote add origin ../origin.git",
				Done: func(sb *Sandbox, _ Attempt) bool { return sb.GitOK("remote", "get-url", "origin") },
			},
			{
				Goal: "Push for the first time",
				Text: "Upload main and link it to the remote branch with `git push -u origin main`. Thanks to `-u`, future pushes and pulls need no arguments.",
				Hint: "git push -u origin main",
				Done: func(sb *Sandbox, _ Attempt) bool {
					return sb.Git("rev-parse", "--abbrev-ref", "main@{u}") == "origin/main"
				},
			},
			{
				Goal: "Fetch a teammate's work",
				Prepare: func(sb *Sandbox) error {
					return sb.script("",
						"git clone -q origin.git teammate",
						`cd teammate && echo "Contributing guide" > CONTRIBUTING.md && git add . && git commit -qm "Add contributing guide" && git push -q`)
				},
				Text: "Your teammate just pushed a commit. Download it with `git fetch` (safe: your files don't change), then `git status` tells you you're behind.",
				Hint: "git fetch && git status",
				Done: func(sb *Sandbox, _ Attempt) bool {
					return sb.commitCount("HEAD..origin/main") >= 1 || sb.exists("CONTRIBUTING.md")
				},
			},
			{
				Goal: "Pull it in",
				Text: "Integrate the new commit into your branch with `git pull`. CONTRIBUTING.md appears in your folder.",
				Hint: "git pull",
				Done: func(sb *Sandbox, _ Attempt) bool {
					return sb.exists("CONTRIBUTING.md") && sb.commitCount("HEAD..origin/main") == 0
				},
			},
		},
	},
}
