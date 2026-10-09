package main

import (
	"context"

	"github.com/gammons/slk/internal/avatar"
	"github.com/gammons/slk/internal/debuglog"
	"github.com/gammons/slk/internal/ui/workspace"
)

type teamIconer interface {
	TeamIcon(ctx context.Context) (string, error)
}

// loadWorkspaceIcon fetches the workspace icon for the rail into the
// avatar cache. Its AvatarReadyMsg (keyed workspace.IconKey) redraws
// the rail; until then, or on failure, the rail shows initials.
func loadWorkspaceIcon(ctx context.Context, client teamIconer, teamID string, avatars *avatar.Cache) {
	url, err := client.TeamIcon(ctx)
	if err != nil {
		debuglog.General("workspace icon %s: %v", teamID, err)
		return
	}
	avatars.Preload(workspace.IconKey(teamID), url)
}
