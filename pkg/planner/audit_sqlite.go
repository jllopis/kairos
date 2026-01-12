package planner

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

// SQLiteAuditStore persists audit events in SQLite.
type SQLiteAuditStore struct {
	db *sql.DB
}

// NewSQLiteAuditStore creates a SQLite-backed audit store and ensures schema.
func NewSQLiteAuditStore(db *sql.DB) (*SQLiteAuditStore, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	if err := ensurePlannerAuditSchema(db); err != nil {
		return nil, err
	}
	return &SQLiteAuditStore{db: db}, nil
}

// Record stores a single audit event.
func (s *SQLiteAuditStore) Record(ctx context.Context, event AuditEvent) error {
	output, err := encodeAuditOutput(event.Output)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO planner_audit_events (
			graph_id, run_id, node_id, node_type, status, output_json, error_text, started_at, finished_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.GraphID,
		event.RunID,
		event.NodeID,
		event.NodeType,
		event.Status,
		string(output),
		event.Error,
		normalizeAuditTime(event.StartedAt),
		normalizeAuditTime(event.FinishedAt),
	)
	return err
}

// List returns audit events matching the filter.
func (s *SQLiteAuditStore) List(ctx context.Context, filter AuditFilter) ([]AuditEvent, error) {
	query := `
		SELECT graph_id, run_id, node_id, node_type, status, output_json, error_text, started_at, finished_at
		FROM planner_audit_events
	`
	var args []any
	where := ""
	addFilter := func(clause string, value any) {
		if where == "" {
			where = " WHERE " + clause
		} else {
			where += " AND " + clause
		}
		args = append(args, value)
	}
	if filter.GraphID != "" {
		addFilter("graph_id = ?", filter.GraphID)
	}
	if filter.NodeID != "" {
		addFilter("node_id = ?", filter.NodeID)
	}
	if filter.Status != "" {
		addFilter("status = ?", filter.Status)
	}
	query += where + " ORDER BY started_at ASC, rowid ASC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var (
			event      AuditEvent
			outputJSON string
			started    sql.NullTime
			finished   sql.NullTime
		)
		if err := rows.Scan(
			&event.GraphID,
			&event.RunID,
			&event.NodeID,
			&event.NodeType,
			&event.Status,
			&outputJSON,
			&event.Error,
			&started,
			&finished,
		); err != nil {
			return nil, err
		}
		if outputJSON != "" {
			if out, err := decodeAuditOutput([]byte(outputJSON)); err == nil {
				event.Output = out
			}
		}
		if started.Valid {
			event.StartedAt = started.Time
		}
		if finished.Valid {
			event.FinishedAt = finished.Time
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}

func ensurePlannerAuditSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS planner_audit_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			graph_id TEXT NOT NULL,
			run_id TEXT,
			node_id TEXT NOT NULL,
			node_type TEXT NOT NULL,
			status TEXT NOT NULL,
			output_json TEXT,
			error_text TEXT,
			started_at TIMESTAMP,
			finished_at TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_planner_audit_graph ON planner_audit_events(graph_id);
		CREATE INDEX IF NOT EXISTS idx_planner_audit_node ON planner_audit_events(node_id);
		CREATE INDEX IF NOT EXISTS idx_planner_audit_status ON planner_audit_events(status);
	`)
	return err
}

// ensure audit schema uses SQLite timestamps in UTC.
func init() {
	_ = time.UTC
}
