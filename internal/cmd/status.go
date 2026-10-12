package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"gitwise/internal/advisor"
	"gitwise/internal/git"
	"gitwise/internal/ui"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:     "status",
		Aliases: []string{"st", "next"},
		Short:   "Explain the repository state in plain words and suggest next steps",
		RunE:    func(cmd *cobra.Command, args []string) error { return runStatus() },
	})
}

const maxListed = 12

func runStatus() error {
	s, err := requireRepo()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(ui.Title.Render("gitwise status") + "\n\n")
	b.WriteString(branchLine(s) + "\n")
	if s.LastCommit != "" {
		b.WriteString(label("Last commit") + s.LastCommit + "\n")
	}
	if s.Operation != "" {
		b.WriteString("\n" + ui.Warn.Bold(true).Render("⚠ A "+s.Operation+" is in progress.") + " " +
			ui.Muted.Render("Finish or abort it before doing anything else.") + "\n")
	}

	section(&b, ui.Danger, "Conflicts", "both sides changed these — fix them by hand", plainList(s.Conflicted, "both modified"))
	section(&b, ui.Success, "Staged", "will be included in your next commit", changeList(s.Staged))
	section(&b, ui.Warn, "Not staged", "changed, but not added yet", changeList(s.Unstaged))
	section(&b, ui.Muted, "Untracked", "new files Git doesn't track yet", plainList(s.Untracked, "new"))

	if s.Clean() && s.Operation == "" {
		b.WriteString("\n" + ui.Success.Render("✓ Working tree clean — nothing to commit.") + "\n")
	}
	if s.Stashes > 0 {
		b.WriteString("\n" + label("Stashes") + fmt.Sprintf("%d saved (git stash list)", s.Stashes) + "\n")
	}

	b.WriteString("\n" + ui.Heading.Render("What next?") + "\n")
	b.WriteString(renderSuggestions(advisor.Suggest(s)))
	fmt.Println(b.String())
	return nil
}

func label(s string) string {
	return ui.Muted.Width(13).Render(s)
}

func branchLine(s *git.Status) string {
	var where string
	switch {
	case s.Detached:
		where = ui.Danger.Render("detached HEAD") + ui.Muted.Render(" (you are not on any branch)")
	case s.Initial:
		where = ui.Bold.Render(s.Branch) + ui.Muted.Render(" (no commits yet)")
	default:
		where = ui.Bold.Render(s.Branch)
	}
	if s.Upstream != "" {
		where += ui.Muted.Render(" ⇄ ") + s.Upstream
		switch {
		case s.Ahead == 0 && s.Behind == 0:
			where += " " + ui.Success.Render("✓ up to date")
		default:
			if s.Ahead > 0 {
				where += " " + ui.Warn.Render(fmt.Sprintf("↑%d to push", s.Ahead))
			}
			if s.Behind > 0 {
				where += " " + ui.Warn.Render(fmt.Sprintf("↓%d to pull", s.Behind))
			}
		}
	} else if !s.Detached && !s.Initial {
		where += ui.Muted.Render(" (not on the remote yet)")
	}
	return label("Branch") + where
}

func section(b *strings.Builder, color lipgloss.Style, title, desc string, lines []string) {
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s %s %s\n", color.Render("●"), ui.Bold.Render(fmt.Sprintf("%s (%d)", title, len(lines))), ui.Muted.Render("— "+desc))
	for i, l := range lines {
		if i == maxListed {
			fmt.Fprintf(b, "    %s\n", ui.Muted.Render(fmt.Sprintf("…and %d more", len(lines)-maxListed)))
			break
		}
		fmt.Fprintf(b, "    %s\n", l)
	}
}

func changeList(cs []git.Change) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = ui.Muted.Width(13).Render(c.Describe()) + c.Label()
	}
	return out
}

func plainList(paths []string, kind string) []string {
	out := make([]string, len(paths))
	for i, p := range paths {
		out[i] = ui.Muted.Width(13).Render(kind) + p
	}
	return out
}

func renderSuggestions(sugs []advisor.Suggestion) string {
	rows := make([][2]string, len(sugs))
	for i, s := range sugs {
		rows[i] = [2]string{s.Command, s.Reason}
	}
	return ui.CommandTable(rows)
}
