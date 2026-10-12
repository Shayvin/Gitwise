package ui

import (
	"errors"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ErrCancelled is returned when the user quits a prompt without choosing.
var ErrCancelled = errors.New("cancelled")

// Item is an entry in a Picker.
type Item struct {
	Name  string
	Desc  string
	Extra string // additional text matched by the filter
	index int
}

func (i Item) Title() string       { return i.Name }
func (i Item) Description() string { return i.Desc }
func (i Item) FilterValue() string { return i.Name + " " + i.Extra }

type pickerModel struct {
	list   list.Model
	chosen int
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width-2, msg.Height-1)
	case tea.KeyMsg:
		if msg.String() == "enter" && m.list.FilterState() != list.Filtering {
			if it, ok := m.list.SelectedItem().(Item); ok {
				m.chosen = it.index
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickerModel) View() string {
	return lipgloss.NewStyle().Margin(1, 1, 0, 1).Render(m.list.View())
}

// NewList builds a styled bubbles list from items.
func NewList(title string, items []Item) list.Model {
	li := make([]list.Item, len(items))
	for i := range items {
		items[i].index = i
		li[i] = items[i]
	}
	d := list.NewDefaultDelegate()
	d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(ColorAccent).BorderForeground(ColorAccent)
	d.Styles.SelectedDesc = d.Styles.SelectedDesc.Foreground(ColorMuted).BorderForeground(ColorAccent)

	l := list.New(li, d, 0, 0)
	l.Title = title
	l.Styles.Title = Title
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select"))}
	}
	return l
}

// Pick shows a full-screen filterable list and returns the chosen index.
func Pick(title string, items []Item) (int, error) {
	m := pickerModel{list: NewList(title, items), chosen: -1}
	res, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return -1, err
	}
	if c := res.(pickerModel).chosen; c >= 0 {
		return c, nil
	}
	return -1, ErrCancelled
}
