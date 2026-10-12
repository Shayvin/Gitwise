package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"gitwise/internal/lessons"
)

type learnState int

const (
	stateMenu learnState = iota
	stateLesson
	stateComplete
)

type cmdDoneMsg struct {
	line, out string
	failed    bool
}

type logEntry struct {
	kind string // "cmd", "out", "ok", "info", "err"
	text string
}

type learnModel struct {
	state    learnState
	progress *lessons.Progress
	menu     list.Model

	lesson *lessons.Lesson
	sb     *lessons.Sandbox
	step   int
	hint   bool
	busy   bool
	log    []logEntry

	input   textinput.Model
	view    viewport.Model
	bar     progress.Model
	spin    spinner.Model
	history []string
	histPos int

	width, height int
}

// RunLearn starts the interactive lesson browser. If startID names a
// lesson, it is opened directly.
func RunLearn(startID string) error {
	m := newLearnModel()
	if startID != "" {
		for i := range lessons.All {
			if lessons.All[i].ID == startID {
				m.startLesson(i)
			}
		}
	}
	res, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if lm, ok := res.(*learnModel); ok && lm.sb != nil {
		lm.sb.Remove()
	}
	return err
}

func newLearnModel() *learnModel {
	in := textinput.New()
	in.Prompt = Heading.Render("$ ")
	in.Placeholder = "type a command, e.g. git status"
	in.CharLimit = 500

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = Heading

	m := &learnModel{
		progress: lessons.LoadProgress(),
		input:    in,
		bar:      progress.New(progress.WithGradient("#D9542F", "#F2A541"), progress.WithoutPercentage()),
		spin:     sp,
		view:     viewport.New(80, 10),
	}
	m.menu = NewList("Git lessons", m.menuItems())
	return m
}

func (m *learnModel) menuItems() []Item {
	items := make([]Item, len(lessons.All))
	for i, l := range lessons.All {
		name := l.Title
		if m.progress.Completed[l.ID] {
			name += "  " + Success.Render("✓ done")
		}
		items[i] = Item{Name: name, Desc: fmt.Sprintf("%s · %d steps", l.Summary, len(l.Steps))}
	}
	return items
}

func (m *learnModel) Init() tea.Cmd { return textinput.Blink }

func (m *learnModel) startLesson(i int) {
	if m.sb != nil {
		m.sb.Remove()
	}
	m.lesson = &lessons.All[i]
	m.step, m.hint, m.log, m.busy = 0, false, nil, false
	m.state = stateLesson

	sb, err := lessons.NewSandbox()
	if err != nil {
		m.addLog("err", err.Error())
		return
	}
	m.sb = sb
	m.addLog("info", "Sandbox ready at "+sb.Dir+"\nEverything here is a throwaway copy — experiment freely!")
	if m.lesson.Setup != nil {
		if err := m.lesson.Setup(sb); err != nil {
			m.addLog("err", err.Error())
		}
	}
	m.prepareStep()
	m.input.Focus()
	m.input.SetValue("")
}

func (m *learnModel) prepareStep() {
	if p := m.lesson.Steps[m.step].Prepare; p != nil {
		if err := p(m.sb); err != nil {
			m.addLog("err", err.Error())
		}
	}
}

func (m *learnModel) addLog(kind, text string) {
	m.log = append(m.log, logEntry{kind, strings.TrimRight(text, "\n")})
	m.refreshLog()
}

func (m *learnModel) refreshLog() {
	w := max(m.view.Width-1, 20)
	var b strings.Builder
	for _, e := range m.log {
		var s string
		switch e.kind {
		case "cmd":
			s = Heading.Render("$ ") + Bold.Render(e.text)
		case "ok":
			s = Success.Bold(true).Render("✓ " + e.text)
		case "info":
			s = Muted.Render(e.text)
		case "err":
			s = Danger.Render(e.text)
		default:
			s = e.text
		}
		b.WriteString(lipgloss.NewStyle().Width(w).Render(s))
		b.WriteString("\n")
	}
	m.view.SetContent(b.String())
	m.view.GotoBottom()
}

func (m *learnModel) runLine(line string) tea.Cmd {
	sb := m.sb
	return func() tea.Msg {
		out, err := sb.Shell("project", line)
		return cmdDoneMsg{line: line, out: out, failed: err != nil}
	}
}

func (m *learnModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.menu.SetSize(msg.Width-2, msg.Height-1)
		m.layout()
		return m, nil

	case spinner.TickMsg:
		if !m.busy {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd

	case cmdDoneMsg:
		m.busy = false
		if strings.TrimSpace(msg.out) != "" {
			m.addLog("out", msg.out)
		}
		step := m.lesson.Steps[m.step]
		if step.Done(m.sb, lessons.Attempt{Line: msg.line, Output: msg.out}) {
			m.advance()
		} else if msg.failed && strings.TrimSpace(msg.out) == "" {
			m.addLog("err", "command failed")
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		switch m.state {
		case stateMenu:
			return m.updateMenu(msg)
		case stateComplete:
			return m.updateComplete(msg)
		case stateLesson:
			return m.updateLesson(msg)
		}
	}

	if m.state == stateMenu {
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *learnModel) advance() {
	m.addLog("ok", m.lesson.Steps[m.step].Goal)
	m.step++
	m.hint = false
	if m.step >= len(m.lesson.Steps) {
		m.state = stateComplete
		if err := m.progress.MarkDone(m.lesson.ID); err != nil {
			m.addLog("err", "could not save progress: "+err.Error())
		}
		m.menu.SetItems(toListItems(m.menuItems()))
		return
	}
	m.prepareStep()
	m.layout()
}

func toListItems(items []Item) []list.Item {
	out := make([]list.Item, len(items))
	for i := range items {
		items[i].index = i
		out[i] = items[i]
	}
	return out
}

func (m *learnModel) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" && m.menu.FilterState() != list.Filtering {
		if it, ok := m.menu.SelectedItem().(Item); ok {
			m.startLesson(it.index)
			m.layout()
			return m, textinput.Blink
		}
	}
	var cmd tea.Cmd
	m.menu, cmd = m.menu.Update(msg)
	return m, cmd
}

func (m *learnModel) updateComplete(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "n":
		next := m.lessonIndex() + 1
		if next < len(lessons.All) {
			m.startLesson(next)
			m.layout()
			return m, textinput.Blink
		}
		m.state = stateMenu
	case "esc", "m":
		m.state = stateMenu
	case "q":
		return m, tea.Quit
	}
	return m, nil
}

func (m *learnModel) lessonIndex() int {
	for i := range lessons.All {
		if &lessons.All[i] == m.lesson {
			return i
		}
	}
	return -1
}

func (m *learnModel) updateLesson(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateMenu
		return m, nil
	case "tab":
		m.hint = !m.hint
		m.layout()
		return m, nil
	case "ctrl+n":
		m.addLog("info", "Skipped: "+m.lesson.Steps[m.step].Goal)
		m.advance()
		return m, nil
	case "pgup", "pgdown":
		var cmd tea.Cmd
		m.view, cmd = m.view.Update(msg)
		return m, cmd
	case "up":
		if m.histPos > 0 {
			m.histPos--
			m.input.SetValue(m.history[m.histPos])
			m.input.CursorEnd()
		}
		return m, nil
	case "down":
		if m.histPos < len(m.history)-1 {
			m.histPos++
			m.input.SetValue(m.history[m.histPos])
		} else {
			m.histPos = len(m.history)
			m.input.SetValue("")
		}
		return m, nil
	case "enter":
		if m.busy {
			return m, nil
		}
		line := strings.TrimSpace(m.input.Value())
		m.input.SetValue("")
		if line == "" {
			return m, nil
		}
		m.history = append(m.history, line)
		m.histPos = len(m.history)
		switch {
		case line == "clear":
			m.log = nil
			m.refreshLog()
			return m, nil
		case line == "exit" || line == "quit":
			m.state = stateMenu
			return m, nil
		case line == "hint":
			m.hint = true
			m.layout()
			return m, nil
		case line == "cd" || strings.HasPrefix(line, "cd "):
			m.addLog("cmd", line)
			m.addLog("info", "No need to cd — every command runs inside the lesson's project folder.")
			return m, nil
		}
		m.addLog("cmd", line)
		m.busy = true
		return m, tea.Batch(m.runLine(line), m.spin.Tick)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// layout sizes the output viewport to whatever space the header leaves.
func (m *learnModel) layout() {
	if m.width == 0 || m.lesson == nil {
		return
	}
	m.bar.Width = min(30, m.width/3)
	m.input.Width = m.width - 6
	m.view.Width = m.width - 2
	m.view.Height = max(m.height-lipgloss.Height(m.header())-3, 3)
	m.refreshLog()
}

func (m *learnModel) header() string {
	total := len(m.lesson.Steps)
	cur := min(m.step+1, total)
	top := lipgloss.JoinHorizontal(lipgloss.Center,
		Title.Render(m.lesson.Title), "  ",
		m.bar.ViewAs(float64(m.step)/float64(total)), " ",
		Muted.Render(fmt.Sprintf("step %d/%d", cur, total)))

	if m.step >= total {
		return top
	}
	step := m.lesson.Steps[m.step]
	body := Heading.Render("▶ "+step.Goal) + "\n" + Inline(step.Text)
	if m.hint {
		body += "\n\n" + Warn.Render("Hint: ") + Code.Render(step.Hint)
	}
	box := Box.Width(m.width - 4).Render(body)
	return top + "\n" + box
}

func (m *learnModel) View() string {
	switch m.state {
	case stateMenu:
		return lipgloss.NewStyle().Margin(1, 1, 0, 1).Render(m.menu.View())
	case stateComplete:
		return m.completeView()
	}
	if m.width == 0 {
		return ""
	}
	prompt := m.input.View()
	if m.busy {
		prompt = m.spin.View() + " running…"
	}
	help := Muted.Render("enter run • ↑↓ history • tab hint • ctrl+n skip • pgup/pgdn scroll • esc lessons • ctrl+c quit")
	return lipgloss.JoinVertical(lipgloss.Left, m.header(), m.view.View(), prompt, help)
}

func (m *learnModel) completeView() string {
	next := "enter next lesson"
	if m.lessonIndex() == len(lessons.All)-1 {
		next = "enter lesson list"
	}
	body := Success.Bold(true).Render("🎉 Lesson complete: "+m.lesson.Title) + "\n\n" +
		"You practised: " + m.lesson.Summary + ".\n\n" +
		Muted.Render(next+" • esc lesson list • q quit")
	box := Box.BorderForeground(ColorSuccess).Padding(1, 3).Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}
