package server

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/encoding/protojson"
)

const approvalTable = "a2a_approvals"

var approvalJSON = protojson.MarshalOptions{EmitUnpopulated: true}

// SQLiteApprovalStore persists approvals in a SQLite database.
type SQLiteApprovalStore struct {
	db *sql.DB
}

// NewSQLiteApprovalStore creates a SQLite-backed approval store and ensures schema.
func NewSQLiteApprovalStore(db *sql.DB) (*SQLiteApprovalStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if err := ensureSQLiteSchema(db); err != nil {
		return nil, err
	}
	return &SQLiteApprovalStore{db: db}, nil
}

// Create inserts an approval record.
func (s *SQLiteApprovalStore) Create(ctx context.Context, record ApprovalRecord) (*ApprovalRecord, error) {
	if record.TaskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	if record.Message == nil {
		return nil, fmt.Errorf("message is required")
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	if record.Status == "" {
		record.Status = ApprovalStatusPending
	}
	now := time.Now().UTC()
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.UpdatedAt = now
	payload, err := approvalJSON.Marshal(record.Message)
	if err != nil {
		return nil, err
	}
	expiresAt := int64(0)
	if !record.ExpiresAt.IsZero() {
		expiresAt = record.ExpiresAt.UnixMilli()
	}
	_, err = s.db.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (id, task_id, context_id, status, reason, created_at, updated_at, message_json, expires_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)", approvalTable),
		record.ID, record.TaskID, record.ContextID, string(record.Status), record.Reason, record.CreatedAt.UnixMilli(), record.UpdatedAt.UnixMilli(), payload, expiresAt)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, record.ID)
}

// Get returns an approval record by id.
func (s *SQLiteApprovalStore) Get(ctx context.Context, id string) (*ApprovalRecord, error) {
	row := s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT id, task_id, context_id, status, reason, created_at, updated_at, message_json, expires_at FROM %s WHERE id = ?", approvalTable),
		id,
	)
	var (
		record      ApprovalRecord
		status      string
		createdAtMs int64
		updatedAtMs int64
		expiresAtMs int64
		messageJSON []byte
	)
	if err := row.Scan(&record.ID, &record.TaskID, &record.ContextID, &status, &record.Reason, &createdAtMs, &updatedAtMs, &messageJSON, &expiresAtMs); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("approval %q not found", id)
		}
		return nil, err
	}
	record.Status = ApprovalStatus(status)
	record.CreatedAt = time.UnixMilli(createdAtMs).UTC()
	record.UpdatedAt = time.UnixMilli(updatedAtMs).UTC()
	if expiresAtMs > 0 {
		record.ExpiresAt = time.UnixMilli(expiresAtMs).UTC()
	}
	if len(messageJSON) > 0 {
		message := &a2av1.Message{}
		if err := protojson.Unmarshal(messageJSON, message); err != nil {
			return nil, err
		}
		record.Message = message
	}
	return &record, nil
}

// List returns approvals matching the filter.
func (s *SQLiteApprovalStore) List(ctx context.Context, filter ApprovalFilter) ([]*ApprovalRecord, error) {
	where := "1=1"
	args := make([]any, 0)
	if filter.TaskID != "" {
		where += " AND task_id = ?"
		args = append(args, filter.TaskID)
	}
	if filter.ContextID != "" {
		where += " AND context_id = ?"
		args = append(args, filter.ContextID)
	}
	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, string(filter.Status))
	}
	if !filter.ExpiringBefore.IsZero() {
		where += " AND expires_at > 0 AND expires_at <= ?"
		args = append(args, filter.ExpiringBefore.UnixMilli())
	}
	limit := ""
	if filter.Limit > 0 {
		limit = fmt.Sprintf(" LIMIT %d", filter.Limit)
	}
	query := fmt.Sprintf("SELECT id, task_id, context_id, status, reason, created_at, updated_at, message_json, expires_at FROM %s WHERE %s ORDER BY updated_at DESC%s", approvalTable, where, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*ApprovalRecord, 0)
	for rows.Next() {
		var (
			record      ApprovalRecord
			status      string
			createdAtMs int64
			updatedAtMs int64
			expiresAtMs int64
			messageJSON []byte
		)
		if err := rows.Scan(&record.ID, &record.TaskID, &record.ContextID, &status, &record.Reason, &createdAtMs, &updatedAtMs, &messageJSON, &expiresAtMs); err != nil {
			return nil, err
		}
		record.Status = ApprovalStatus(status)
		record.CreatedAt = time.UnixMilli(createdAtMs).UTC()
		record.UpdatedAt = time.UnixMilli(updatedAtMs).UTC()
		if expiresAtMs > 0 {
			record.ExpiresAt = time.UnixMilli(expiresAtMs).UTC()
		}
		if len(messageJSON) > 0 {
			message := &a2av1.Message{}
			if err := protojson.Unmarshal(messageJSON, message); err != nil {
				return nil, err
			}
			record.Message = message
		}
		out = append(out, &record)
	}
	return out, rows.Err()
}

// UpdateStatus updates approval status and reason.
func (s *SQLiteApprovalStore) UpdateStatus(ctx context.Context, id string, status ApprovalStatus, reason string) (*ApprovalRecord, error) {
	_, err := s.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET status = ?, reason = ?, updated_at = ? WHERE id = ?", approvalTable),
		string(status), reason, time.Now().UTC().UnixMilli(), id)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}
