package cache

import "fmt"

// SetSectionCollapsed records that the user collapsed or expanded a
// sidebar section. sectionKey is the Slack section ID, or the section
// name for config-defined sections. Both states are stored, so an
// expanded section that defaults to collapsed stays expanded.
func (db *DB) SetSectionCollapsed(workspaceID, sectionKey string, collapsed bool) error {
	v := 0
	if collapsed {
		v = 1
	}
	_, err := db.conn.Exec(`
		INSERT INTO sidebar_collapse (workspace_id, section_key, collapsed)
		VALUES (?, ?, ?)
		ON CONFLICT(workspace_id, section_key)
		DO UPDATE SET collapsed = excluded.collapsed`,
		workspaceID, sectionKey, v,
	)
	if err != nil {
		return fmt.Errorf("saving section collapse: %w", err)
	}
	return nil
}

// SectionCollapse returns the workspace's saved section states, by
// section key. Sections the user never toggled are absent.
func (db *DB) SectionCollapse(workspaceID string) (map[string]bool, error) {
	rows, err := db.conn.Query(`
		SELECT section_key, collapsed FROM sidebar_collapse WHERE workspace_id = ?`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying section collapse: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var key string
		var v int
		if err := rows.Scan(&key, &v); err != nil {
			return nil, fmt.Errorf("scanning section collapse: %w", err)
		}
		out[key] = v != 0
	}
	return out, rows.Err()
}
