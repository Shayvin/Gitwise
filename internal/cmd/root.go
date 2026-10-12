// Package cmd wires gitwise's subcommands together.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/x/term"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"gitwise/internal/git"
	"gitwise/internal/ui"
)

var rootCmd = &cobra.Command{
	Use:   "gitwise",
	Short: "A friendly mentor that helps you learn and use the git CLI",
	Long: ui.Banner() + `

gitwise explains what git is doing, suggests what to do next, warns you
before destructive commands and teaches git with hands-on lessons.

Run gitwise without arguments for an interactive menu.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMenu()
	},
}

// Execute runs the CLI and exits with a non-zero code on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, ui.ErrCancelled) || errors.Is(err, huh.ErrUserAborted) {
			return
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			os.Exit(exit.ExitCode())
		}
		fmt.Fprintln(os.Stderr, ui.Danger.Render("✗ ")+err.Error())
		os.Exit(1)
	}
}

type menuEntry struct {
	name, desc string
	run        func() error
}

func runMenu() error {
	entries := []menuEntry{
		{"📍 Status", "where am I and what should I do next?", func() error { return runStatus() }},
		{"🎓 Learn", "hands-on lessons in a safe sandbox", func() error { return ui.RunLearn("") }},
		{"✍️  Commit", "guided commit with message checks", func() error { return runCommit() }},
		{"⏪ Undo", "fix a mistake — step-by-step recipes", func() error { return runUndo() }},
		{"🔍 Explain", "break down any git command", func() error {
			var line string
			err := huh.NewInput().Title("Which git command should I explain?").
				Placeholder("git reset --hard HEAD~1").Value(&line).Run()
			if err != nil {
				return err
			}
			return explain(splitCommandLine(line))
		}},
		{"📋 Cheatsheet", "searchable list of common commands", func() error { return runCheatsheet(false) }},
		{"📖 Glossary", "what do HEAD, origin, upstream... mean?", func() error { return printGlossary(nil) }},
		{"🩺 Doctor", "check your git setup", func() error { return runDoctor() }},
	}
	items := make([]ui.Item, len(entries))
	for i, e := range entries {
		items[i] = ui.Item{Name: e.name, Desc: e.desc}
	}
	i, err := ui.Pick("gitwise — what do you want to do?", items)
	if err != nil {
		return err
	}
	return entries[i].run()
}

// --- shared helpers ---

func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return "."
	}
	return d
}

func requireRepo() (*git.Status, error) {
	s, err := git.GetStatus(cwd())
	if errors.Is(err, git.ErrNotRepo) {
		return nil, fmt.Errorf("this folder is not a git repository\n  Start one with %s or download one with %s",
			ui.Code.Render("git init"), ui.Code.Render("git clone <url>"))
	}
	return s, err
}

func interactive() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stdin.Fd())
}

// width returns a comfortable text width for the terminal.
func width() int {
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || w <= 0 {
		return 80
	}
	return min(w-4, 100)
}

// runGit runs git attached to the user's terminal.
func runGit(args ...string) error {
	c := exec.Command("git", args...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// splitCommandLine splits a command line into words, honoring quotes.
func splitCommandLine(s string) []string {
	var out []string
	var cur []rune
	var quote rune
	inWord := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur = append(cur, r)
			}
		case r == '"' || r == '\'':
			quote, inWord = r, true
		case r == ' ' || r == '\t':
			if inWord {
				out = append(out, string(cur))
				cur, inWord = cur[:0], false
			}
		default:
			cur, inWord = append(cur, r), true
		}
	}
	if inWord {
		out = append(out, string(cur))
	}
	return out
}
