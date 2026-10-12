package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitwise/internal/git"
	"gitwise/internal/ui"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "commit",
		Aliases: []string{"ci"},
		Short:   "Guided commit: pick files, write a good message, review, commit",
		RunE:    func(cmd *cobra.Command, args []string) error { return runCommit() },
	})
}

var commitTypes = []huh.Option[string]{
	huh.NewOption("feat      a new feature", "feat"),
	huh.NewOption("fix       a bug fix", "fix"),
	huh.NewOption("docs      documentation only", "docs"),
	huh.NewOption("refactor  code change that neither fixes a bug nor adds a feature", "refactor"),
	huh.NewOption("test      adding or fixing tests", "test"),
	huh.NewOption("style     formatting, whitespace (no code change)", "style"),
	huh.NewOption("perf      performance improvement", "perf"),
	huh.NewOption("chore     tooling, dependencies, build", "chore"),
	huh.NewOption("none      plain message, no prefix", ""),
}

func runCommit() error {
	s, err := requireRepo()
	if err != nil {
		return err
	}
	if s.Operation != "" && s.Operation != "merge" {
		return fmt.Errorf("a %s is in progress — finish it with git %s --continue first", s.Operation, s.Operation)
	}
	if len(s.Conflicted) > 0 {
		return fmt.Errorf("resolve the conflicts in %s first (then git add them)", strings.Join(s.Conflicted, ", "))
	}

	// Every changed path, with staged ones preselected.
	var opts []huh.Option[string]
	staged := map[string]bool{}
	for _, c := range s.Staged {
		staged[c.Path] = true
		opts = append(opts, huh.NewOption(c.Label()+ui.Success.Render("  staged, "+c.Describe()), c.Path).Selected(true))
	}
	for _, c := range s.Unstaged {
		if !staged[c.Path] {
			opts = append(opts, huh.NewOption(c.Label()+ui.Warn.Render("  "+c.Describe()), c.Path))
		}
	}
	for _, p := range s.Untracked {
		opts = append(opts, huh.NewOption(p+ui.Muted.Render("  new, untracked"), p))
	}
	if len(opts) == 0 {
		fmt.Println(ui.Success.Render("✓ Nothing to commit — your working tree is clean."))
		return nil
	}

	var (
		files            []string
		kind, scope, sum string
		body             string
	)
	kind = "feat" // preselect; "" would match the "none" option
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which changes belong in this commit?").
				Description("space to toggle, enter to confirm. One commit = one logical change.").
				Options(opts...).Value(&files).Height(min(len(opts)+2, 15)).
				Validate(func(v []string) error {
					if len(v) == 0 {
						return errors.New("select at least one file")
					}
					return nil
				}),
		),
		huh.NewGroup(
			huh.NewSelect[string]().Title("What kind of change is it?").
				Description("Conventional Commits prefixes make history easy to scan.").
				Options(commitTypes...).Value(&kind),
			huh.NewInput().Title("Scope (optional)").Description("area of the code, e.g. auth, api, ui").Value(&scope),
		),
		huh.NewGroup(
			huh.NewInput().Title("Summary").
				Description("Imperative mood: \"Add login form\", not \"Added\". Aim for ≤ 50 characters.").
				Value(&sum).Validate(validateSummary),
			huh.NewText().Title("Body (optional)").
				Description("Explain WHY, not what — the diff already shows what.").
				Value(&body).Lines(5),
		),
	)
	if err := form.Run(); err != nil {
		return err
	}

	msg := strings.TrimSpace(sum)
	if kind != "" {
		prefix := kind
		if scope = strings.TrimSpace(scope); scope != "" {
			prefix += "(" + scope + ")"
		}
		msg = prefix + ": " + msg
	}
	body = strings.TrimSpace(body)

	preview := ui.Bold.Render(msg)
	if body != "" {
		preview += "\n\n" + body
	}
	preview += "\n\n" + ui.Muted.Render(fmt.Sprintf("%d file(s): %s", len(files), strings.Join(files, ", ")))
	fmt.Println(ui.Box.Render(preview))
	for _, tip := range lintSummary(strings.TrimSpace(sum)) {
		fmt.Println(ui.Warn.Render("  tip: ") + tip)
	}

	ok := true
	if err := huh.NewConfirm().Title("Create this commit?").Value(&ok).Run(); err != nil {
		return err
	}
	if !ok {
		fmt.Println(ui.Muted.Render("Aborted — nothing was staged or committed."))
		return nil
	}

	if err := applySelection(s, files, staged); err != nil {
		return err
	}
	args := []string{"commit", "-m", msg}
	if body != "" {
		args = append(args, "-m", body)
	}
	if err := runGit(args...); err != nil {
		return err
	}
	fmt.Println(ui.Success.Render("✓ Committed!") + ui.Muted.Render(" Share it with: git push   ·   Changed your mind? gitwise undo"))
	return nil
}

// applySelection makes the index contain exactly the selected paths' changes.
func applySelection(s *git.Status, files []string, staged map[string]bool) error {
	dir := cwd()
	selected := map[string]bool{}
	for _, f := range files {
		selected[f] = true
	}
	var unstage []string
	for p := range staged {
		if !selected[p] {
			unstage = append(unstage, p)
		}
	}
	if len(unstage) > 0 {
		args := append([]string{"restore", "--staged", "--"}, unstage...)
		if s.Initial {
			args = append([]string{"rm", "--cached", "-r", "-q", "--"}, unstage...)
		}
		if _, err := git.Run(dir, args...); err != nil {
			return err
		}
	}
	// Paths that are fully staged already need no add (and a staged deletion
	// would make `git add` fail with "pathspec did not match").
	pending := map[string]bool{}
	for _, c := range s.Unstaged {
		pending[c.Path] = true
	}
	for _, p := range s.Untracked {
		pending[p] = true
	}
	var add []string
	for _, f := range files {
		if pending[f] {
			add = append(add, f)
		}
	}
	if len(add) == 0 {
		return nil
	}
	_, err := git.Run(dir, append([]string{"add", "--"}, add...)...)
	return err
}

func validateSummary(s string) error {
	s = strings.TrimSpace(s)
	switch {
	case s == "":
		return errors.New("a summary is required")
	case len(s) > 72:
		return fmt.Errorf("%d characters — keep it under 72 (details go in the body)", len(s))
	}
	return nil
}

var notPastTense = map[string]bool{"embed": true, "need": true, "speed": true, "feed": true, "seed": true, "bring": true, "string": true, "ping": true}

// lintSummary returns style suggestions for a commit summary.
func lintSummary(s string) []string {
	var tips []string
	if len(s) > 50 {
		tips = append(tips, fmt.Sprintf("summary is %d characters; ≤ 50 reads best in git log --oneline", len(s)))
	}
	if strings.HasSuffix(s, ".") {
		tips = append(tips, "summaries usually don't end with a period")
	}
	if first := strings.ToLower(strings.Fields(s + " x")[0]); !notPastTense[first] && (strings.HasSuffix(first, "ed") || strings.HasSuffix(first, "ing")) {
		tips = append(tips, fmt.Sprintf("%q → try the imperative mood (\"Fix\", \"Add\", \"Update\")", first))
	}
	if strings.HasPrefix(strings.ToLower(s), "wip") {
		tips = append(tips, "WIP commits are fine locally — squash them before sharing")
	}
	return tips
}
