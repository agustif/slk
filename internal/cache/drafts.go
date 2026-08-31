package cache

import (
	"fmt"
	"time"
)

// Draft is one parked compose snapshot, keyed per workspace.
type Draft struct {
	WorkspaceID   string
	Key           string
	Text          string
	SlackID       string
	LastUpdatedTS string
	UpdatedAt     int64
}

func (db *DB) UpsertDraft(d Draft) error {
	if d.WorkspaceID == "" || d.Key == "" {
		return fmt.Errorf("upserting draft: empty workspace or key")
	}
	if d.UpdatedAt == 0 {
		d.UpdatedAt = time.Now().Unix()
	}
	_, err := db.conn.Exec(`
		INSERT INTO drafts (workspace_id, draft_key, text, slack_id, last_updated_ts, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace_id, draft_key) DO UPDATE SET
			text=excluded.text,
			slack_id=CASE WHEN excluded.slack_id = '' THEN drafts.slack_id ELSE excluded.slack_id END,
			last_updated_ts=CASE WHEN excluded.last_updated_ts = '' THEN drafts.last_updated_ts ELSE excluded.last_updated_ts END,
			updated_at=excluded.updated_at
	`, d.WorkspaceID, d.Key, d.Text, d.SlackID, d.LastUpdatedTS, d.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upserting draft: %w", err)
	}
	return nil
}

func (db *DB) GetDraft(workspaceID, key string) (Draft, error) {
	var d Draft
	err := db.conn.QueryRow(`
		SELECT workspace_id, draft_key, text, slack_id, last_updated_ts, updated_at
		FROM drafts WHERE workspace_id = ? AND draft_key = ?
	`, workspaceID, key).Scan(&d.WorkspaceID, &d.Key, &d.Text, &d.SlackID, &d.LastUpdatedTS, &d.UpdatedAt)
	if err != nil {
		return d, fmt.Errorf("getting draft: %w", err)
	}
	return d, nil
}

func (db *DB) DeleteDraft(workspaceID, key string) error {
	_, err := db.conn.Exec(`DELETE FROM drafts WHERE workspace_id = ? AND draft_key = ?`, workspaceID, key)
	if err != nil {
		return fmt.Errorf("deleting draft: %w", err)
	}
	return nil
}

func (db *DB) ListDrafts(workspaceID string) ([]Draft, error) {
	rows, err := db.conn.Query(`
		SELECT workspace_id, draft_key, text, slack_id, last_updated_ts, updated_at
		FROM drafts WHERE workspace_id = ?
	`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("listing drafts: %w", err)
	}
	defer rows.Close()
	var out []Draft
	for rows.Next() {
		var d Draft
		if err := rows.Scan(&d.WorkspaceID, &d.Key, &d.Text, &d.SlackID, &d.LastUpdatedTS, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning draft: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
