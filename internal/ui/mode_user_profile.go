// internal/ui/mode_user_profile.go
//
// User-profile-dialog key handler. K, esc and q all close the modal
// and return to Normal mode; every other key is swallowed, since the
// dialog is read-only and has no scroll state of its own.
package ui

import (
	tea "charm.land/bubbletea/v2"
)

func handleUserProfileMode(a *App, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "K", "esc", "q":
		a.userProfile.Close()
		a.SetMode(ModeNormal)
	}
	return nil
}
