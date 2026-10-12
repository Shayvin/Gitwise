package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"gitwise/internal/advisor"
	"gitwise/internal/git"
	"gitwise/internal/knowledge"
	"gitwise/internal/ui"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "run <git args...>",
		Aliases: []string{"safe", "do"},
		Short:   "Run a git command with a safety check for destructive operations",
		Long: `Runs a git command, but first checks it for dangerous patterns
(force push, reset --hard, clean -f, ...). Risky commands are explained and
need confirmation; for some, gitwise can create a backup branch first.

Tip: alias g="gitwise run" to get this protection all the time.`,
		Example:            "  gitwise run push --force\n  gitwise run reset --hard HEAD~2",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}
			return safeRun(args)
		},
	})
}

// commands after which a "what next?" hint is useful
var hintAfter = map[string]bool{
	"add": true, "commit": true, "merge": true, "pull": true, "stash": true,
	"switch": true, "checkout": true, "reset": true, "rebase": true, "restore": true, "init": true,
}

func safeRun(args []string) error {
	if args[0] == "git" {
		args = args[1:]
	}
	sub, _ := knowledge.SplitArgs(args)
	warnings := knowledge.Analyze(args)

	if knowledge.MaxRisk(warnings) >= knowledge.RiskMedium {
		fmt.Println(ui.Code.Render("git " + strings.Join(quoteArgs(args), " ")))
		for _, w := range warnings {
			fmt.Println(ui.WarningBox(w))
		}
		if !interactive() {
			return fmt.Errorf("refusing to run a risky command without a terminal to confirm — use git directly")
		}

		var backup []string
		for _, w := range warnings {
			if w.Backup != nil {
				backup = w.Backup
			}
		}
		const (
			optRun    = "run"
			optBackup = "backup"
			optCancel = "cancel"
		)
		opts := []huh.Option[string]{huh.NewOption("Cancel", optCancel)}
		if backup != nil && git.IsRepo(cwd()) {
			opts = append(opts, huh.NewOption("Create backup branch "+backup[1]+", then run", optBackup))
		}
		opts = append(opts, huh.NewOption("Run it anyway", optRun))

		choice := optCancel
		if err := huh.NewSelect[string]().Title("What do you want to do?").Options(opts...).Value(&choice).Run(); err != nil {
			return err
		}
		switch choice {
		case optCancel:
			fmt.Println(ui.Muted.Render("Nothing was changed."))
			return nil
		case optBackup:
			if _, err := git.Run(cwd(), backup...); err != nil {
				return fmt.Errorf("could not create backup: %w", err)
			}
			fmt.Println(ui.Success.Render("✓ Backup saved as branch "+backup[1]) +
				ui.Muted.Render("  (restore with: git reset --hard "+backup[1]+")"))
		}
	} else {
		for _, w := range warnings {
			fmt.Println(ui.RiskBadge(w.Risk) + " " + ui.Muted.Render(w.Title+" — "+w.Explanation))
		}
	}

	if err := runGit(args...); err != nil {
		if c, ok := knowledge.Lookup(sub); ok {
			fmt.Println(ui.Muted.Render("\nNeed help? gitwise explain " + c.Name))
		}
		return err
	}

	if hintAfter[sub] {
		if s, err := git.GetStatus(cwd()); err == nil {
			if sugs := advisor.Suggest(s); len(sugs) > 0 {
				fmt.Println("\n" + ui.Heading.Render("Next:"))
				fmt.Print(renderSuggestions(sugs[:min(2, len(sugs))]))
			}
		}
	}
	return nil
}
