package ui

import tea "charm.land/bubbletea/v2"

// loadSectionCollapse applies the active workspace's saved sidebar
// section states. Called when a workspace becomes active, before the
// last-viewed channel is restored.
func (a *App) loadSectionCollapse() {
	if a.channels == nil {
		return
	}
	a.sidebar.SetCollapseState(a.channels.SectionCollapse(a.activeTeamID))
}

// toggleSectionCollapse toggles the section header under the sidebar
// cursor and saves the new state, reporting false when the cursor is
// not on a header. Only these explicit toggles are saved: sections
// that open because navigation jumped into them collapse again on the
// next start.
func (a *App) toggleSectionCollapse() (tea.Cmd, bool) {
	if !a.sidebar.ToggleCollapseSelected() {
		return nil, false
	}
	key, ok := a.sidebar.IsSectionHeaderSelected()
	if !ok || a.channels == nil || a.activeTeamID == "" {
		return nil, true
	}
	collapsed := a.sidebar.IsCollapsed(key)
	channels, team := a.channels, a.activeTeamID
	return func() tea.Msg {
		channels.SaveSectionCollapse(team, key, collapsed)
		return nil
	}, true
}
