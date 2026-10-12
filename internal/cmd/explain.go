package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"gitwise/internal/git"
	"gitwise/internal/knowledge"
	"gitwise/internal/ui"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "explain [git] <command> [args...]",
		Aliases: []string{"what", "x"},
		Short:   "Break down a git command word by word",
		Example: "  gitwise explain git reset --hard HEAD~1\n  gitwise explain push -u origin main",
		// Pass git's flags through untouched instead of parsing them ourselves.
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			if len(args) == 1 && strings.Contains(args[0], " ") {
				args = splitCommandLine(args[0]) // gitwise explain "git log --oneline"
			}
			return explain(args)
		},
	})
}

func explain(args []string) error {
	sub, rest := knowledge.SplitArgs(args)
	if sub == "" {
		printCommandIndex()
		return nil
	}
	c, ok := knowledge.Lookup(sub)
	if !ok {
		if alias := git.Config(cwd(), "alias."+sub); alias != "" {
			fmt.Printf("%s is one of your aliases for %s\n\n", ui.Code.Render(sub), ui.Code.Render("git "+alias))
			if strings.HasPrefix(alias, "!") {
				return nil // shell alias, nothing more to break down
			}
			return explain(append(splitCommandLine(alias), rest...))
		}
		msg := fmt.Sprintf("I don't know %q yet.", sub)
		if guess := knowledge.Closest(sub); guess != "" {
			msg += fmt.Sprintf(" Did you mean %s?", ui.Code.Render("git "+guess))
		}
		fmt.Println(ui.Warn.Render(msg))
		fmt.Println(ui.Muted.Render("Full documentation: git help " + sub))
		return nil
	}

	w := width()
	var b strings.Builder
	b.WriteString(ui.Code.Render("git "+strings.Join(append([]string{sub}, quoteArgs(rest)...), " ")) + "\n\n")
	b.WriteString(ui.Title.Render(c.Name) + " " + ui.Bold.Render(c.Summary) + "\n")
	if c.Details != "" {
		b.WriteString(ui.Wrap(ui.Inline(c.Details), w) + "\n")
	}

	if len(rest) > 0 {
		b.WriteString("\n" + ui.Heading.Render("Piece by piece") + "\n")
		for _, p := range breakDown(c, rest) {
			fmt.Fprintf(&b, "  %s %s\n", ui.Code.Render(p[0]), p[1])
		}
	}

	if warnings := knowledge.Analyze(append([]string{sub}, rest...)); len(warnings) > 0 {
		b.WriteString("\n")
		for _, wr := range warnings {
			b.WriteString(ui.WarningBox(wr) + "\n")
		}
	} else if len(rest) > 0 || len(c.Flags) == 0 {
		b.WriteString("\n" + ui.Success.Render("✓ Nothing destructive here.") + "\n")
	}

	if len(rest) == 0 && len(c.Flags) > 0 {
		b.WriteString("\n" + ui.Heading.Render("Useful options") + "\n")
		for _, f := range c.Flags {
			fmt.Fprintf(&b, "  %s %s\n", ui.Bold.Width(24).Render(strings.Join(f.Names, ", ")), f.Desc)
		}
	}
	if len(c.Examples) > 0 {
		b.WriteString("\n" + ui.Heading.Render("Examples") + "\n")
		rows := make([][2]string, len(c.Examples))
		for i, e := range c.Examples {
			rows[i] = [2]string{e.Cmd, e.Desc}
		}
		b.WriteString(ui.CommandTable(rows))
	}
	if len(c.Related) > 0 {
		b.WriteString("\n" + ui.Muted.Render("Related: "+strings.Join(c.Related, ", ")+" · Full docs: git help "+c.Name) + "\n")
	}
	fmt.Print(b.String())
	return nil
}

// breakDown pairs each argument with an explanation.
func breakDown(c *knowledge.Command, args []string) [][2]string {
	var out [][2]string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			out = append(out, [2]string{a, "separator: everything after this is a file path"})
			for _, p := range args[i+1:] {
				out = append(out, [2]string{p, "file path"})
			}
			break
		}
		if f := c.FindFlag(a); f != nil {
			out = append(out, [2]string{a, f.Desc})
			if f.TakesValue && !strings.Contains(a, "=") && i+1 < len(args) {
				i++
				out = append(out, [2]string{quoteArg(args[i]), ui.Muted.Render("↳ value for " + a)})
			}
			continue
		}
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && len(a) > 2 {
			if combined := splitShortFlags(c, a); combined != nil {
				out = append(out, combined...)
				continue
			}
		}
		if strings.HasPrefix(a, "-") && a != "-" {
			out = append(out, [2]string{a, ui.Muted.Render("an option gitwise doesn't know — see git help " + c.Name)})
			continue
		}
		out = append(out, [2]string{quoteArg(a), describeArg(c.Name, a)})
	}
	return out
}

func splitShortFlags(c *knowledge.Command, a string) [][2]string {
	var out [][2]string
	for _, r := range a[1:] {
		f := c.FindFlag("-" + string(r))
		if f == nil {
			return nil
		}
		out = append(out, [2]string{"-" + string(r), f.Desc + ui.Muted.Render(" (part of "+a+")")})
	}
	return out
}

var (
	reAncestor = regexp.MustCompile(`^(.+?)~(\d*)$`)
	reParent   = regexp.MustCompile(`^(.+?)(\^+)$`)
	reReflog   = regexp.MustCompile(`^(.*)@\{(\d+)\}$`)
	reHash     = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
)

func describeArg(sub, a string) string {
	switch {
	case a == "HEAD" || a == "@":
		return "HEAD — the commit you currently have checked out"
	case a == ".":
		return "the current folder — every file in it"
	case a == "-":
		return "the branch you were on before"
	case a == "@{u}" || a == "@{upstream}":
		return "the upstream (remote) branch of the current branch"
	case a == "origin":
		return "origin — the default name of the remote you cloned from"
	case strings.HasPrefix(a, "<") && strings.HasSuffix(a, ">"):
		return "placeholder — replace with your own " + strings.Trim(a, "<>")
	case strings.Contains(a, "..."):
		l, r, _ := strings.Cut(a, "...")
		return fmt.Sprintf("commits on either %s or %s, but not both", orHead(l), orHead(r))
	case strings.Contains(a, ".."):
		l, r, _ := strings.Cut(a, "..")
		return fmt.Sprintf("commits in %s that are not in %s", orHead(r), orHead(l))
	}
	if m := reAncestor.FindStringSubmatch(a); m != nil {
		n := m[2]
		if n == "" {
			n = "1"
		}
		return fmt.Sprintf("%s commit(s) before %s", n, m[1])
	}
	if m := reParent.FindStringSubmatch(a); m != nil && !strings.HasSuffix(m[1], "@{") {
		return fmt.Sprintf("%d generation(s) before %s (first parent)", len(m[2]), m[1])
	}
	if m := reReflog.FindStringSubmatch(a); m != nil {
		if m[1] == "stash" {
			return "stash entry #" + m[2] + " (see git stash list)"
		}
		return fmt.Sprintf("where %s was %s moves ago (from the reflog)", orHead(m[1]), m[2])
	}
	if sub == "stash" {
		switch a {
		case "push", "pop", "apply", "list", "drop", "show", "clear", "branch":
			return "stash action: " + a
		}
	}
	if sub == "remote" {
		switch a {
		case "add", "remove", "rename", "set-url", "get-url", "show", "prune":
			return "remote action: " + a
		}
	}

	dir := cwd()
	if git.IsRepo(dir) {
		switch {
		case git.RefExists(dir, "refs/heads/"+a):
			return "a local branch in this repository"
		case git.RefExists(dir, "refs/remotes/"+a):
			return "a remote-tracking branch (your copy of a branch on the server)"
		case git.RefExists(dir, "refs/tags/"+a):
			return "a tag in this repository"
		case reHash.MatchString(a) && git.RefExists(dir, a):
			subject, _ := git.Run(dir, "log", "-1", "--format=%s", a)
			return fmt.Sprintf("commit %s — %q", a, subject)
		}
	}
	if _, err := os.Stat(a); err == nil {
		return "a file or folder in the current directory"
	}
	if reHash.MatchString(a) {
		return "looks like a commit hash"
	}
	if strings.HasPrefix(a, "origin/") {
		return "a remote-tracking branch on origin"
	}
	if strings.Contains(a, "://") || strings.HasPrefix(a, "git@") {
		return "a repository URL"
	}
	return ui.Muted.Render("argument — e.g. a branch name, file path or value")
}

func orHead(s string) string {
	if s == "" {
		return "HEAD"
	}
	return s
}

func quoteArg(a string) string {
	if strings.ContainsAny(a, " \t") {
		return `"` + a + `"`
	}
	return a
}

func quoteArgs(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = quoteArg(a)
	}
	return out
}

func printCommandIndex() {
	fmt.Println(ui.Title.Render("git commands gitwise can explain"))
	for _, cat := range knowledge.Categories {
		fmt.Println("\n" + ui.Heading.Render(cat))
		for _, c := range knowledge.Commands {
			if c.Category == cat {
				fmt.Printf("  %s %s\n", ui.Bold.Width(13).Render(c.Name), ui.Muted.Render(c.Summary))
			}
		}
	}
	fmt.Println("\n" + ui.Muted.Render("Try: gitwise explain git log --oneline --graph"))
}
