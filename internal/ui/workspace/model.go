package workspace

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gammons/slk/internal/ui/styles"
)

type WorkspaceItem struct {
	ID        string
	Name      string
	Initials  string
	HasUnread bool
}

type Model struct {
	items    []WorkspaceItem
	selected int
	version  int64
	// unreadReader returns the set of workspace IDs whose dot should
	// be lit: those with at least one channel their own sidebar would
	// show as unread (mute-filtered) or one unread subscribed thread
	// (what the sidebar's Threads badge counts); see
	// railUnreadWorkspaces in cmd/slk. Set by App via SetUnreadReader;
	// called only by RefreshUnreads, since it queries SQLite on the UI
	// goroutine.
	unreadReader func() []string
	// lastUnread is the set the last RefreshUnreads read, which
	// OtherUnreadCount counts.
	lastUnread []string
	// iconFn returns a workspace's icon rendered IconCols×IconRows, or
	// "" while it is not loaded. nil draws initials only.
	iconFn func(teamID string) string
}

// IconCols × IconRows is the size of a workspace icon in the rail. It
// matches avatar.AvatarCols × avatar.AvatarRows, the size the icon is
// rendered at.
const (
	IconCols = 4
	IconRows = 2
)

// IconKey is the avatar-cache key a workspace icon is stored under, so
// it never collides with a user ID.
func IconKey(teamID string) string { return "team:" + teamID }

// SetIconFunc sets the icon renderer. With one set, every rail item is
// IconRows tall, and items whose icon has not loaded show initials.
func (m *Model) SetIconFunc(fn func(teamID string) string) {
	m.iconFn = fn
	m.dirty()
}

// IconReady redraws the rail after a workspace icon finished loading.
func (m *Model) IconReady() { m.dirty() }

// itemRows is the height of one rail item.
func (m Model) itemRows() int {
	if m.iconFn != nil {
		return IconRows
	}
	return 1
}

// Version returns a counter that increments any time the View() output could
// change.
func (m *Model) Version() int64 { return m.version }

func (m *Model) dirty() { m.version++ }

func New(items []WorkspaceItem, selected int) Model {
	return Model{items: items, selected: selected}
}

func (m *Model) SelectedID() string {
	if len(m.items) == 0 {
		return ""
	}
	return m.items[m.selected].ID
}

// NameByID returns the workspace's display name, or "" if no item with
// the given ID is present. Used by the App to derive the window title's
// initials via WorkspaceInitials.
func (m *Model) NameByID(id string) string {
	for _, item := range m.items {
		if item.ID == id {
			return item.Name
		}
	}
	return ""
}

// OtherUnreadCount returns the number of workspaces with unreads,
// excluding activeID, as of the last RefreshUnreads; 0 before any. It
// does not call the reader itself, so a refresh followed by this call
// reads it once. Deciding what counts is the reader's job,
// not this method's: it applies the sidebar's IsVisiblyUnread predicate
// per workspace and asks the same thread query the Threads badge uses,
// so the title's "+N" and the rail dots agree with each workspace's
// own sidebar. Note the active workspace's "(N)" and $SLK_UNREAD count
// channels only; "+N" counts workspaces, and a workspace whose only
// unread is a thread counts.
func (m *Model) OtherUnreadCount(activeID string) int {
	count := 0
	for _, id := range m.lastUnread {
		if id != activeID {
			count++
		}
	}
	return count
}

func (m *Model) SelectedIndex() int {
	return m.selected
}

func (m *Model) Select(idx int) {
	if idx >= 0 && idx < len(m.items) && m.selected != idx {
		m.selected = idx
		m.dirty()
	}
}

func (m *Model) SetItems(items []WorkspaceItem) {
	m.items = items
	if m.selected >= len(items) {
		m.selected = 0
	}
	m.dirty()
}

func (m *Model) SelectByID(teamID string) {
	for i, item := range m.items {
		if item.ID == teamID {
			if m.selected != i {
				m.selected = i
				m.dirty()
			}
			return
		}
	}
}

func (m *Model) SetUnread(teamID string, hasUnread bool) {
	for i := range m.items {
		if m.items[i].ID == teamID {
			if m.items[i].HasUnread != hasUnread {
				m.items[i].HasUnread = hasUnread
				m.dirty()
			}
			return
		}
	}
}

// SetUnreadReader installs the callback used by RefreshUnreads.
func (m *Model) SetUnreadReader(f func() []string) {
	m.unreadReader = f
	if f == nil {
		m.lastUnread = nil
	}
}

// RefreshUnreads pulls the latest set of workspaces-with-unreads from
// the reader and updates each item's HasUnread field. Called by App
// on ReadStateChangedMsg. No-op if no reader is installed.
func (m *Model) RefreshUnreads() {
	if m.unreadReader == nil {
		return
	}
	m.lastUnread = m.unreadReader()
	set := make(map[string]bool, len(m.items))
	for _, id := range m.lastUnread {
		set[id] = true
	}
	changed := false
	for i := range m.items {
		want := set[m.items[i].ID]
		if m.items[i].HasUnread != want {
			m.items[i].HasUnread = want
			changed = true
		}
	}
	if changed {
		m.dirty()
	}
}

func (m Model) View(height int) string {
	if len(m.items) == 0 {
		return ""
	}

	var rows []string
	for i, item := range m.items {
		var icon string
		if m.iconFn != nil {
			icon = m.iconFn(item.ID)
		}
		if icon != "" {
			rows = append(rows, m.iconItem(icon, i == m.selected, item.HasUnread))
			continue
		}
		var style lipgloss.Style
		if i == m.selected {
			style = styles.WorkspaceActive
		} else {
			style = styles.WorkspaceInactive
		}

		initials := item.Initials
		if item.HasUnread && i != m.selected {
			initials = initials + styles.PresenceOnline.Render("●")
		}
		label := style.Render(initials)
		rows = append(rows, label+strings.Repeat("\n", m.itemRows()-1))
	}

	content := strings.Join(rows, "\n\n")

	// Height/MaxHeight in lipgloss include padding in the total,
	// so use the full height directly. Padding(1,0) adds 1 row
	// top + 1 row bottom inside that total, matching the visual
	// offset of RoundedBorder() on adjacent panels.
	rail := lipgloss.NewStyle().
		Width(6).
		Height(height).
		MaxHeight(height).
		Background(styles.RailBackground).
		Padding(1, 0).
		Align(lipgloss.Center).
		Render(content)

	return rail
}

// iconItem lays out one icon item, 6 columns wide: a bar in the
// accent color marks the active workspace (a background highlight
// would not show through the image), and a dot to the right of the
// icon marks unread.
func (m Model) iconItem(icon string, active, unread bool) string {
	bg := lipgloss.NewStyle().Background(styles.RailBackground)
	left := bg.Render(" ")
	if active {
		left = bg.Foreground(styles.Primary).Render("▌")
	}
	lines := strings.Split(icon, "\n")
	for len(lines) < IconRows {
		lines = append(lines, strings.Repeat(" ", IconCols))
	}
	out := make([]string, IconRows)
	for r := 0; r < IconRows; r++ {
		right := bg.Render(" ")
		if r == 0 && unread && !active {
			right = styles.PresenceOnline.Background(styles.RailBackground).Render("●")
		}
		out[r] = left + lines[r] + right
	}
	return strings.Join(out, "\n")
}

// ClickAt returns the workspace item rendered at rail-local row y,
// or ok=false when the click landed on a padding row, a gap between
// items, or past the last item.
//
// Row layout mirrors View(): Padding(1,0) puts blank padding at row 0,
// and items are "\n\n"-joined so they occupy rows 1, 3, 5, ... with
// blank gap rows at 2, 4, 6, ...; with icons each item is IconRows
// tall, so items start at rows 1, 4, 7, .... There is no horizontal column check
// because the rail has no border and uses its full 6-col width as the
// click target.
func (m Model) ClickAt(y int) (WorkspaceItem, bool) {
	if y < 1 || len(m.items) == 0 {
		return WorkspaceItem{}, false
	}
	rel := y - 1
	stride := m.itemRows() + 1
	if rel%stride >= m.itemRows() {
		return WorkspaceItem{}, false // gap between items
	}
	idx := rel / stride
	if idx < 0 || idx >= len(m.items) {
		return WorkspaceItem{}, false
	}
	return m.items[idx], true
}

// Width is the rail's column count. With a single workspace there is
// nothing to switch between, so the rail is not drawn and takes no room.
func (m Model) Width() int {
	if len(m.items) <= 1 {
		return 0
	}
	return 6 // 6 content, no border
}

func WorkspaceInitials(name string) string {
	words := strings.Fields(name)
	switch len(words) {
	case 0:
		return "?"
	case 1:
		if len(words[0]) >= 2 {
			return strings.ToUpper(words[0][:2])
		}
		return strings.ToUpper(words[0])
	default:
		return strings.ToUpper(fmt.Sprintf("%c%c", words[0][0], words[1][0]))
	}
}
