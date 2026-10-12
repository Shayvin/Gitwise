// Package ui contains gitwise's terminal styling and interactive components.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"gitwise/internal/knowledge"
)

var (
	ColorAccent  = lipgloss.AdaptiveColor{Light: "#B8411F", Dark: "#EA7A3A"}
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#56D364"}
	ColorWarning = lipgloss.AdaptiveColor{Light: "#9A5B12", Dark: "#F2A541"}
	ColorDanger  = lipgloss.AdaptiveColor{Light: "#CF222E", Dark: "#FF7B72"}
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#6E6059", Dark: "#A8988F"}
	ColorCodeBg  = lipgloss.AdaptiveColor{Light: "#F3EBE3", Dark: "#2E2420"}
)

var (
	Title   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(ColorAccent).Padding(0, 1)
	Heading = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	Muted   = lipgloss.NewStyle().Foreground(ColorMuted)
	Success = lipgloss.NewStyle().Foreground(ColorSuccess)
	Warn    = lipgloss.NewStyle().Foreground(ColorWarning)
	Danger  = lipgloss.NewStyle().Foreground(ColorDanger)
	Bold    = lipgloss.NewStyle().Bold(true)
	Code    = lipgloss.NewStyle().Foreground(ColorAccent).Background(ColorCodeBg).Padding(0, 1)
	Box     = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(ColorAccent).Padding(0, 1)
)

// Banner is the gitwise logo line.
func Banner() string {
	return Title.Render("gitwise") + " " + Muted.Render("your friendly git mentor")
}

// RiskStyle returns the style used for a risk level.
func RiskStyle(r knowledge.Risk) lipgloss.Style {
	switch r {
	case knowledge.RiskHigh:
		return Danger.Bold(true)
	case knowledge.RiskMedium:
		return Warn.Bold(true)
	case knowledge.RiskLow:
		return Warn
	}
	return Success
}

// RiskBadge renders a small colored label like "● high risk".
func RiskBadge(r knowledge.Risk) string {
	return RiskStyle(r).Render("● " + r.String())
}

// WarningBox renders a danger warning with its explanation and alternatives.
func WarningBox(w knowledge.Warning) string {
	color := ColorWarning
	if w.Risk == knowledge.RiskHigh {
		color = ColorDanger
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s\n", RiskBadge(w.Risk), Bold.Render(w.Title))
	b.WriteString(Inline(w.Explanation))
	if w.Safer != "" {
		fmt.Fprintf(&b, "\n%s %s", Success.Render("Safer:"), Code.Render(w.Safer))
	}
	return Box.BorderForeground(color).Width(76).Render(b.String())
}

// CommandTable renders aligned "command  description" rows.
func CommandTable(rows [][2]string) string {
	w := 0
	for _, r := range rows {
		w = max(w, lipgloss.Width(r[0]))
	}
	w = min(w, 46)
	var b strings.Builder
	for _, r := range rows {
		pad := max(w-lipgloss.Width(r[0]), 0)
		fmt.Fprintf(&b, "  %s%s  %s\n", Code.Render(r[0]), strings.Repeat(" ", pad), Muted.Render(r[1]))
	}
	return b.String()
}

// Bullet renders an indented list item.
func Bullet(sym lipgloss.Style, s string) string {
	return "  " + sym.Render("•") + " " + s
}

// Wrap soft-wraps text to width.
func Wrap(s string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(s)
}

// Inline renders `backtick` spans in s as code.
func Inline(s string) string {
	parts := strings.Split(s, "`")
	for i := 1; i < len(parts); i += 2 {
		parts[i] = Code.Render(parts[i])
	}
	return strings.Join(parts, "")
}
