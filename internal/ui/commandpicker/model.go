// Package commandpicker is the inline autocomplete for slash commands,
// opened by a "/" at the start of a message. It mirrors channelpicker:
// the compose model owns the trigger and the query, this package the
// list, the filter and the rendering.
package commandpicker

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/gammons/slk/internal/text"
	"github.com/gammons/slk/internal/ui/styles"
)

// MaxVisible is the number of rows the dropdown shows.
const MaxVisible = 6

// Command is one slash command. Name carries the leading slash.
type Command struct {
	Name    string
	Desc    string
	Usage   string
	AppName string // owning app or service; "" for built-ins
}

type Model struct {
	commands []Command
	filtered []Command
	query    string
	selected int
	visible  bool
}

func New() Model { return Model{} }

func (m *Model) SetCommands(commands []Command) {
	m.commands = commands
	if m.visible {
		m.filter()
	}
}

func (m *Model) Open() {
	m.visible = true
	m.query = ""
	m.selected = 0
	m.filter()
}

func (m *Model) Close() {
	m.visible = false
	m.query = ""
	m.selected = 0
	m.filtered = nil
}

func (m *Model) IsVisible() bool { return m.visible }

// SetQuery filters by the text typed after the slash.
func (m *Model) SetQuery(q string) {
	m.query = q
	m.selected = 0
	m.filter()
}

func (m *Model) Filtered() []Command { return m.filtered }

func (m *Model) MoveUp() {
	if m.selected > 0 {
		m.selected--
	}
}

func (m *Model) MoveDown() {
	if m.selected < len(m.filtered)-1 {
		m.selected++
	}
}

// Select returns the highlighted command, or nil when nothing matches.
func (m *Model) Select() *Command {
	if m.selected < 0 || m.selected >= len(m.filtered) {
		return nil
	}
	c := m.filtered[m.selected]
	return &c
}

// filter keeps the commands whose name (without the slash) starts with
// the query, then those whose name or app contains it.
func (m *Model) filter() {
	q := text.Fold(m.query)
	var prefix, rest []Command
	for _, c := range m.commands {
		name := text.Fold(strings.TrimPrefix(c.Name, "/"))
		switch {
		case strings.HasPrefix(name, q):
			prefix = append(prefix, c)
		case strings.Contains(name, q) || (q != "" && strings.Contains(text.Fold(c.AppName), q)):
			rest = append(rest, c)
		}
	}
	m.filtered = append(prefix, rest...)
	if len(m.filtered) > MaxVisible {
		m.filtered = m.filtered[:MaxVisible]
	}
}

func (m *Model) View(width int) string {
	if !m.visible || len(m.filtered) == 0 {
		return ""
	}
	inner := width - 2
	if inner < 10 {
		inner = 10
	}
	var rows []string
	for i, c := range m.filtered {
		indicator := "  "
		nameStyle := lipgloss.NewStyle().Foreground(styles.TextPrimary)
		if i == m.selected {
			indicator = lipgloss.NewStyle().Foreground(styles.Accent).Render("▌ ")
			nameStyle = nameStyle.Bold(true)
		}
		label := c.Name
		if c.Usage != "" {
			label += " " + c.Usage
		}
		detail := c.Desc
		if c.AppName != "" {
			if detail != "" {
				detail = c.AppName + " · " + detail
			} else {
				detail = c.AppName
			}
		}
		row := indicator + nameStyle.Render(label)
		if detail != "" {
			row += "  " + styles.Timestamp.Render(detail)
		}
		rows = append(rows, ansi.Truncate(row, inner, "…"))
	}
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Primary).
		Background(styles.SurfaceDark).
		Width(width - 2).
		Render(strings.Join(rows, "\n"))
}
