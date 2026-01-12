package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	a2av1 "github.com/jllopis/kairos/pkg/a2a/types"
	"google.golang.org/protobuf/encoding/protojson"

	_ "modernc.org/sqlite"
)

const (
	taskTable       = "a2a_tasks"
	pushConfigTable = "a2a_push_configs"
)

var (
	taskJSON      = protojson.MarshalOptions{EmitUnpopulated: true}
	taskUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// SQLiteTaskStore persists A2A tasks in a SQLite database.
type SQLiteTaskStore struct {
	db *sql.DB
}

// SQLitePushConfigStore persists push notification configs in a SQLite database.
type SQLitePushConfigStore struct {
	db *sql.DB
}

// NewSQLiteTaskStore creates a SQLite-backed task store and ensures schema.
func NewSQLiteTaskStore(db *sql.DB) (*SQLiteTaskStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if err := ensureSQLiteSchema(db); err != nil {
		return nil, err
	}
	return &SQLiteTaskStore{db: db}, nil
}

// NewSQLitePushConfigStore creates a SQLite-backed push config store and ensures schema.
func NewSQLitePushConfigStore(db *sql.DB) (*SQLitePushConfigStore, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	if err := ensureSQLiteSchema(db); err != nil {
		return nil, err
	}
	return &SQLitePushConfigStore{db: db}, nil
}

func ensureSQLiteSchema(db *sql.DB) error {
	stmts := []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			context_id TEXT NOT NULL,
			status_state INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			task_json BLOB NOT NULL
		);`, taskTable),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_context ON %s(context_id);`, taskTable, taskTable),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_status ON %s(status_state);`, taskTable, taskTable),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_updated ON %s(updated_at);`, taskTable, taskTable),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			context_id TEXT NOT NULL,
			status TEXT NOT NULL,
			reason TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			message_json BLOB NOT NULL,
			expires_at INTEGER NOT NULL DEFAULT 0
		);`, approvalTable),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_task ON %s(task_id);`, approvalTable, approvalTable),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_status ON %s(status);`, approvalTable, approvalTable),
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			task_id TEXT NOT NULL,
			config_id TEXT NOT NULL,
			config_json BLOB NOT NULL,
			PRIMARY KEY(task_id, config_id)
		);`, pushConfigTable),
		fmt.Sprintf(`CREATE INDEX IF NOT EXISTS idx_%s_task ON %s(task_id);`, pushConfigTable, pushConfigTable),
	}
	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	if err := ensureApprovalExpiryColumn(db); err != nil {
		return err
	}
	return nil
}

func ensureApprovalExpiryColumn(db *sql.DB) error {
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN expires_at INTEGER NOT NULL DEFAULT 0", approvalTable))
	if err == nil {
		return nil
	}
	if strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return nil
	}
	return err
}

// CreateTask persists a new task seeded from the incoming message.
func (s *SQLiteTaskStore) CreateTask(ctx context.Context, message *a2av1.Message) (*a2av1.Task, error) {
	if message == nil {
		return nil, fmt.Errorf("message is nil")
	}
	taskID := uuid.NewString()
	contextID := message.ContextId
	if contextID == "" {
		contextID = uuid.NewString()
	}

	msg := cloneMessage(message)
	msg.TaskId = taskID
	msg.ContextId = contextID
	status := newStatus(a2av1.TaskState_TASK_STATE_SUBMITTED, msg)
	task := &a2av1.Task{
		Id:        taskID,
		ContextId: contextID,
		Status:    status,
		History:   []*a2av1.Message{msg},
	}

	payload, err := marshalTask(task)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().UnixMilli()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx,
		fmt.Sprintf("INSERT INTO %s (id, context_id, status_state, updated_at, task_json) VALUES (?, ?, ?, ?, ?)", taskTable),
		taskID, contextID, int32(status.State), now, payload)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return cloneTask(task), nil
}

// AppendHistory appends a message to the task history.
func (s *SQLiteTaskStore) AppendHistory(ctx context.Context, taskID string, message *a2av1.Message) error {
	if message == nil {
		return fmt.Errorf("message is nil")
	}
	task, err := s.getTask(ctx, taskID)
	if err != nil {
		return err
	}
	task.History = append(task.History, cloneMessage(message))
	return s.updateTask(ctx, task)
}

// UpdateStatus updates the persisted task status.
func (s *SQLiteTaskStore) UpdateStatus(ctx context.Context, taskID string, status *a2av1.TaskStatus) error {
	if status == nil {
		return fmt.Errorf("status is nil")
	}
	task, err := s.getTask(ctx, taskID)
	if err != nil {
		return err
	}
	task.Status = status
	return s.updateTask(ctx, task)
}

// AddArtifacts appends artifacts to a persisted task.
func (s *SQLiteTaskStore) AddArtifacts(ctx context.Context, taskID string, artifacts []*a2av1.Artifact) error {
	if len(artifacts) == 0 {
		return nil
	}
	task, err := s.getTask(ctx, taskID)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		if artifact == nil {
			continue
		}
		task.Artifacts = append(task.Artifacts, artifact)
	}
	return s.updateTask(ctx, task)
}

// GetTask returns a task with optional history/artifact filtering.
func (s *SQLiteTaskStore) GetTask(ctx context.Context, taskID string, historyLength int32, includeArtifacts bool) (*a2av1.Task, error) {
	task, err := s.getTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return filterTask(task, historyLength, includeArtifacts), nil
}

// ListTasks lists tasks using the provided filter and pagination settings.
func (s *SQLiteTaskStore) ListTasks(ctx context.Context, filter TaskFilter) ([]*a2av1.Task, int, error) {
	pageSize := int(filter.PageSize)
	if pageSize <= 0 {
		pageSize = 50
	}
	offset := 0
	if filter.PageToken != "" {
		parsed, err := parsePageToken(filter.PageToken)
		if err != nil {
			return nil, 0, errInvalidPageToken
		}
		offset = parsed
	}
	if offset < 0 {
		return nil, 0, errInvalidPageToken
	}

	where, args := buildTaskFilter(filter)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s%s", taskTable, where)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if offset >= total {
		return []*a2av1.Task{}, total, nil
	}

	query := fmt.Sprintf(`SELECT task_json FROM %s%s ORDER BY updated_at DESC, id ASC LIMIT ? OFFSET ?`, taskTable, where)
	args = append(args, pageSize, offset)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []*a2av1.Task
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, 0, err
		}
		task, err := unmarshalTask(payload)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, filterTask(task, filter.HistoryLength, filter.IncludeArtifacts))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// CancelTask updates the task state to cancelled when possible.
func (s *SQLiteTaskStore) CancelTask(ctx context.Context, taskID string) (*a2av1.Task, error) {
	task, err := s.getTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if isTerminalState(task.GetStatus().GetState()) && task.GetStatus().GetState() != a2av1.TaskState_TASK_STATE_CANCELLED {
		return task, nil
	}
	task.Status = newStatus(a2av1.TaskState_TASK_STATE_CANCELLED, task.GetStatus().GetMessage())
	if err := s.updateTask(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *SQLiteTaskStore) getTask(ctx context.Context, taskID string) (*a2av1.Task, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT task_json FROM %s WHERE id = ?", taskTable), taskID).Scan(&payload)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("task %q not found", taskID)
		}
		return nil, err
	}
	return unmarshalTask(payload)
}

func (s *SQLiteTaskStore) updateTask(ctx context.Context, task *a2av1.Task) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	payload, err := marshalTask(task)
	if err != nil {
		return err
	}
	now := time.Now().UTC().UnixMilli()
	_, err = s.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET context_id = ?, status_state = ?, updated_at = ?, task_json = ? WHERE id = ?", taskTable),
		task.ContextId, int32(task.GetStatus().GetState()), now, payload, task.Id)
	return err
}

// Set stores a push notification config for a task.
func (s *SQLitePushConfigStore) Set(ctx context.Context, taskID, configID string, config *a2av1.TaskPushNotificationConfig) (*a2av1.TaskPushNotificationConfig, error) {
	if taskID == "" || configID == "" {
		return nil, fmt.Errorf("task id and config id are required")
	}
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}
	payload, err := taskJSON.Marshal(config)
	if err != nil {
		return nil, err
	}
	_, err = s.db.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO %s (task_id, config_id, config_json) VALUES (?, ?, ?)
			ON CONFLICT(task_id, config_id) DO UPDATE SET config_json = excluded.config_json`, pushConfigTable),
		taskID, configID, payload)
	if err != nil {
		return nil, err
	}
	return cloneTaskPushNotificationConfig(config), nil
}

// Get returns a push notification config for a task.
func (s *SQLitePushConfigStore) Get(ctx context.Context, taskID, configID string) (*a2av1.TaskPushNotificationConfig, error) {
	var payload []byte
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT config_json FROM %s WHERE task_id = ? AND config_id = ?", pushConfigTable),
		taskID, configID).Scan(&payload)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("config %q not found", configID)
		}
		return nil, err
	}
	var cfg a2av1.TaskPushNotificationConfig
	if err := taskUnmarshal.Unmarshal(payload, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// List lists push notification configs for a task.
func (s *SQLitePushConfigStore) List(ctx context.Context, taskID string, pageSize int32) ([]*a2av1.TaskPushNotificationConfig, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	rows, err := s.db.QueryContext(ctx,
		fmt.Sprintf("SELECT config_json FROM %s WHERE task_id = ? ORDER BY config_id ASC LIMIT ?", pushConfigTable),
		taskID, pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*a2av1.TaskPushNotificationConfig
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var cfg a2av1.TaskPushNotificationConfig
		if err := taskUnmarshal.Unmarshal(payload, &cfg); err != nil {
			return nil, err
		}
		out = append(out, &cfg)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Delete removes a push notification config for a task.
func (s *SQLitePushConfigStore) Delete(ctx context.Context, taskID, configID string) error {
	result, err := s.db.ExecContext(ctx,
		fmt.Sprintf("DELETE FROM %s WHERE task_id = ? AND config_id = ?", pushConfigTable),
		taskID, configID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("config %q not found", configID)
	}
	return nil
}

func buildTaskFilter(filter TaskFilter) (string, []any) {
	var clauses []string
	var args []any
	if filter.ContextID != "" {
		clauses = append(clauses, "context_id = ?")
		args = append(args, filter.ContextID)
	}
	if filter.Status != a2av1.TaskState_TASK_STATE_UNSPECIFIED {
		clauses = append(clauses, "status_state = ?")
		args = append(args, int32(filter.Status))
	}
	if !filter.LastUpdatedAfter.IsZero() {
		clauses = append(clauses, "updated_at >= ?")
		args = append(args, filter.LastUpdatedAfter.UTC().UnixMilli())
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + joinClauses(clauses, " AND "), args
}

func joinClauses(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep + parts[i]
	}
	return out
}

func marshalTask(task *a2av1.Task) ([]byte, error) {
	return taskJSON.Marshal(task)
}

func unmarshalTask(payload []byte) (*a2av1.Task, error) {
	var task a2av1.Task
	if err := taskUnmarshal.Unmarshal(payload, &task); err != nil {
		return nil, err
	}
	return &task, nil
}
