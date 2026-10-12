package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitwise/internal/git"
	"gitwise/internal/knowledge"
	"gitwise/internal/ui"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "undo",
		Aliases: []string{"oops", "fix"},
		Short:   "Pick what went wrong and get (and optionally run) the fix",
		RunE:    func(cmd *cobra.Command, args []string) error { return runUndo() },
	})
}

var rePlaceholder = regexp.MustCompile(`<[^<>]+>`)

func runUndo() error {
	s, err := requireRepo()
	if err != nil {
		return err
	}

	// Recipes relevant to the current state go first.
	var order []int
	for pass := 0; pass < 2; pass++ {
		for i, r := range knowledge.UndoRecipes {
			rel := r.Relevant != nil && r.Relevant(s)
			if rel == (pass == 0) {
				order = append(order, i)
			}
		}
	}
	items := make([]ui.Item, len(order))
	for i, idx := range order {
		r := knowledge.UndoRecipes[idx]
		desc := r.Problem
		if r.Relevant != nil && r.Relevant(s) {
			desc = "★ " + desc
		}
		items[i] = ui.Item{Name: r.Title, Desc: desc, Extra: strings.Join(r.Steps, " ")}
	}
	choice, err := ui.Pick("What do you want to undo?  (★ = matches your repo right now)", items)
	if err != nil {
		return err
	}
	r := knowledge.UndoRecipes[order[choice]]

	fmt.Println(ui.Title.Render(r.Title) + "  " + ui.RiskBadge(r.Risk))
	if r.Explain != "" {
		fmt.Println(ui.Wrap(ui.Inline(r.Explain), width()))
	}
	fmt.Println()
	for i, step := range r.Steps {
		fmt.Printf("  %d. %s\n", i+1, ui.Code.Render(step))
	}
	fmt.Println()

	values, err := fillPlaceholders(r.Steps, s)
	if err != nil {
		return err
	}
	run := false
	title := "Run this now?"
	if len(r.Steps) > 1 {
		title = fmt.Sprintf("Run these %d commands now?", len(r.Steps))
	}
	if err := huh.NewConfirm().Title(title).Affirmative("Run").Negative("No, just show me").Value(&run).Run(); err != nil {
		return err
	}
	if !run {
		fmt.Println(ui.Muted.Render("Copy the commands above whenever you're ready."))
		return nil
	}

	for _, step := range r.Steps {
		args := splitCommandLine(step)
		for i := range args {
			args[i] = rePlaceholder.ReplaceAllStringFunc(args[i], func(p string) string { return values[p] })
		}
		fmt.Println(ui.Heading.Render("$ ") + strings.Join(quoteArgs(args), " "))
		if err := runGit(args[1:]...); err != nil {
			return fmt.Errorf("step failed, stopping here: %w", err)
		}
	}
	fmt.Println(ui.Success.Render("✓ Done.") + ui.Muted.Render(" Check the result with: gitwise status"))
	return nil
}

// fillPlaceholders asks the user for every <placeholder> in steps, offering
// choices from the repository where possible.
func fillPlaceholders(steps []string, s *git.Status) (map[string]string, error) {
	values := map[string]string{}
	for _, step := range steps {
		for _, p := range rePlaceholder.FindAllString(step, -1) {
			if _, done := values[p]; done {
				continue
			}
			var v string
			var field huh.Field
			name := strings.Trim(p, "<>")
			switch {
			case name == "file":
				opts := fileOptions(s)
				if len(opts) == 0 {
					field = huh.NewInput().Title("Which file?").Value(&v)
				} else {
					field = huh.NewSelect[string]().Title("Which file?").Options(opts...).Value(&v)
				}
			case name == "commit" || strings.HasPrefix(name, "hash"):
				if opts := commitOptions(name != "commit"); len(opts) > 0 {
					field = huh.NewSelect[string]().Title("Which commit?").Options(opts...).Value(&v).Height(12)
				} else {
					field = huh.NewInput().Title("Commit hash").Value(&v)
				}
			default:
				field = huh.NewInput().Title("Enter " + name).Value(&v).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("can't be empty")
						}
						return nil
					})
			}
			if err := huh.NewForm(huh.NewGroup(field)).Run(); err != nil {
				return nil, err
			}
			values[p] = strings.TrimSpace(v)
		}
	}
	return values, nil
}

func fileOptions(s *git.Status) []huh.Option[string] {
	seen := map[string]bool{}
	var opts []huh.Option[string]
	addOpt := func(path, kind string) {
		if !seen[path] {
			seen[path] = true
			opts = append(opts, huh.NewOption(path+ui.Muted.Render("  ("+kind+")"), path))
		}
	}
	for _, c := range s.Staged {
		addOpt(c.Path, "staged, "+c.Describe())
	}
	for _, c := range s.Unstaged {
		addOpt(c.Path, c.Describe())
	}
	for _, p := range s.Untracked {
		addOpt(p, "untracked")
	}
	return opts
}

func commitOptions(fromReflog bool) []huh.Option[string] {
	args := []string{"log", "-20", "--format=%h%x09%s (%cr)"}
	if fromReflog {
		args = []string{"reflog", "-20", "--format=%h%x09%gs"}
	}
	out, _ := git.Run(cwd(), args...)
	var opts []huh.Option[string]
	for _, line := range strings.Split(out, "\n") {
		hash, desc, ok := strings.Cut(line, "\t")
		if ok {
			opts = append(opts, huh.NewOption(ui.Bold.Render(hash)+" "+desc, hash))
		}
	}
	return opts
}
