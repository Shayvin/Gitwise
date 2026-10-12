package knowledge

import "strings"

// Term is a glossary entry.
type Term struct {
	Name string
	Def  string
}

// Glossary explains the vocabulary new git users run into.
var Glossary = []Term{
	{"repository (repo)", "A project folder tracked by Git, including its complete history stored in the hidden .git directory."},
	{"working tree", "The actual files you see and edit on disk."},
	{"staging area (index)", "A draft of your next commit. `git add` puts changes here; `git commit` saves exactly what is staged."},
	{"commit", "A saved snapshot of the staged files plus a message, author, date and a link to its parent commit. Identified by a hash like a1b2c3d."},
	{"hash (SHA)", "The unique ID of a commit, e.g. 3f9e1c2. The first 7 characters are usually enough to refer to it."},
	{"branch", "A movable label pointing to a commit. When you commit, the current branch moves forward to the new commit."},
	{"HEAD", "\"You are here\". Points to the branch (or commit) you currently have checked out."},
	{"detached HEAD", "HEAD points directly to a commit instead of a branch. New commits here belong to no branch — create one with `git switch -c <name>` to keep them."},
	{"HEAD~1 / HEAD^", "The commit before HEAD. HEAD~3 means three commits back."},
	{"remote", "Another copy of the repository, usually on a server like GitHub. The default one is named origin."},
	{"origin", "The conventional name of the remote you cloned from."},
	{"remote-tracking branch", "A read-only local copy of a branch on the remote, e.g. origin/main. Updated by `git fetch`."},
	{"upstream", "The remote branch your local branch is linked to. `git push` and `git pull` use it by default."},
	{"ahead / behind", "How many commits your branch has that the upstream doesn't (ahead), and vice versa (behind)."},
	{"fetch", "Download new commits from a remote without changing your files."},
	{"pull", "Fetch plus merge (or rebase) into your current branch."},
	{"push", "Upload your local commits to the remote."},
	{"merge", "Combine the histories of two branches. May create a merge commit with two parents."},
	{"fast-forward", "A merge where your branch simply moves ahead because no diverging commits exist. No merge commit is needed."},
	{"merge conflict", "Both branches changed the same lines. Git marks them with <<<<<<<, ======= and >>>>>>> and asks you to choose."},
	{"rebase", "Replay your commits on top of another commit, producing new commits with a linear history."},
	{"stash", "A shelf for uncommitted changes, so you can get a clean working tree and come back later."},
	{"tracked / untracked", "Tracked files are known to Git (they were committed or staged). Untracked files are new files Git ignores until you add them."},
	{".gitignore", "A file listing patterns of files Git should never track, like node_modules/ or *.log."},
	{"reflog", "A local log of every position HEAD has been in. The place to look when you 'lost' commits."},
	{"tag", "A fixed name for a specific commit, typically a release like v1.2.0."},
	{"cherry-pick", "Copy one commit from elsewhere onto your current branch."},
	{"pull request (PR) / merge request", "Not a Git feature but a hosting feature: a request to merge your branch, with review and discussion."},
	{"fork", "Your own server-side copy of someone else's repository, used to propose changes via pull requests."},
	{"amend", "Replace the most recent commit with a corrected version."},
	{"squash", "Combine several commits into one."},
}

// FindTerms returns glossary entries whose name contains q (case-insensitive).
func FindTerms(q string) []Term {
	q = strings.ToLower(strings.TrimSpace(q))
	var out []Term
	for _, t := range Glossary {
		if strings.Contains(strings.ToLower(t.Name), q) {
			out = append(out, t)
		}
	}
	return out
}
