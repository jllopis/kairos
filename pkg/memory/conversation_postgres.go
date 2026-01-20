// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package memory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
)

var tableNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func sanitizeTableName(table string) (string, error) {
	if table == "" {
		return "", fmt.Errorf("table name is required")
	}
	if !tableNamePattern.MatchString(table) {
		return "", fmt.Errorf("invalid table name %q", table)
	}
	return table, nil
}

// PostgresConversation implements ConversationMemory with PostgreSQL storage.
// Suitable for production deployments with multiple instances.
type PostgresConversation struct {
	db     *sql.DB
	table  string
	config ConversationConfig
}

// PostgresConfig configures the PostgreSQL conversation store.
type PostgresConfig struct {
	// DB is the database connection. Required.
	DB *sql.DB
	// TableName is the table to use. Default: "conversation_messages".
	TableName string
	// ConversationConfig for truncation and TTL.
	ConversationConfig ConversationConfig
}

// NewPostgresConversation creates a new PostgreSQL conversation store.
// Call Initialize() to create the table if it doesn't exist.
func NewPostgresConversation(cfg PostgresConfig) (*PostgresConversation, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database connection is required")
	}

	table := cfg.TableName
	if table == "" {
		table = "conversation_messages"
	}
	table, err := sanitizeTableName(table)
	if err != nil {
		return nil, err
	}

	return &PostgresConversation{
		db:     cfg.DB,
		table:  table,
		config: cfg.ConversationConfig,
	}, nil
}

// Initialize creates the conversation table if it doesn't exist.
func (p *PostgresConversation) Initialize(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY,
			session_id VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL,
			content TEXT NOT NULL,
			tool_call_id VARCHAR(255),
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_%s_session ON %s (session_id);
		CREATE INDEX IF NOT EXISTS idx_%s_created ON %s (created_at);
		CREATE INDEX IF NOT EXISTS idx_%s_session_created ON %s (session_id, created_at, id);
	`, p.table, p.table, p.table, p.table, p.table, p.table, p.table)

	_, err := p.db.ExecContext(ctx, query)
	return err
}

// AppendMessage adds a message to the conversation.
func (p *PostgresConversation) AppendMessage(ctx context.Context, sessionID string, msg ConversationMessage) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.SessionID == "" {
		msg.SessionID = sessionID
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	var metadataJSON []byte
	var err error
	if msg.Metadata != nil {
		metadataJSON, err = json.Marshal(msg.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (id, session_id, role, content, tool_call_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, p.table)

	_, err = p.db.ExecContext(ctx, query,
		msg.ID,
		sessionID,
		msg.Role,
		msg.Content,
		sql.NullString{String: msg.ToolCallID, Valid: msg.ToolCallID != ""},
		metadataJSON,
		msg.CreatedAt,
	)

	return err
}

// GetMessages retrieves all messages for a session.
func (p *PostgresConversation) GetMessages(ctx context.Context, sessionID string) ([]ConversationMessage, error) {
	query := fmt.Sprintf(`
		SELECT id, session_id, role, content, tool_call_id, metadata, created_at
		FROM %s
		WHERE session_id = $1
		ORDER BY created_at ASC
	`, p.table)

	messages, err := p.queryMessages(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}

	// Apply truncation strategy if configured
	if p.config.TruncationStrategy != nil && len(messages) > 0 {
		return p.config.TruncationStrategy.Truncate(ctx, messages)
	}

	return messages, nil
}

// GetRecentMessages retrieves the last N messages for a session.
func (p *PostgresConversation) GetRecentMessages(ctx context.Context, sessionID string, limit int) ([]ConversationMessage, error) {
	// Use subquery to get last N in correct order
	query := fmt.Sprintf(`
		SELECT id, session_id, role, content, tool_call_id, metadata, created_at
		FROM (
			SELECT id, session_id, role, content, tool_call_id, metadata, created_at
			FROM %s
			WHERE session_id = $1
			ORDER BY created_at DESC
			LIMIT $2
		) sub
		ORDER BY created_at ASC
	`, p.table)

	return p.queryMessages(ctx, query, sessionID, limit)
}

// Clear removes all messages for a session.
func (p *PostgresConversation) Clear(ctx context.Context, sessionID string) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE session_id = $1`, p.table)
	_, err := p.db.ExecContext(ctx, query, sessionID)
	return err
}

// DeleteOldMessages removes messages older than the given duration.
func (p *PostgresConversation) DeleteOldMessages(ctx context.Context, sessionID string, olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE session_id = $1 AND created_at < $2
	`, p.table)
	_, err := p.db.ExecContext(ctx, query, sessionID, cutoff)
	return err
}

// DeleteOldSessions removes all messages from sessions inactive for the given duration.
func (p *PostgresConversation) DeleteOldSessions(ctx context.Context, inactiveDuration time.Duration) (int64, error) {
	cutoff := time.Now().Add(-inactiveDuration)
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE session_id IN (
			SELECT session_id
			FROM %s
			GROUP BY session_id
			HAVING MAX(created_at) < $1
		)
	`, p.table, p.table)

	result, err := p.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ListSessions returns all active session IDs.
func (p *PostgresConversation) ListSessions(ctx context.Context) ([]string, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT session_id
		FROM %s
		ORDER BY session_id
	`, p.table)

	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []string
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil {
			return nil, err
		}
		sessions = append(sessions, sessionID)
	}

	return sessions, rows.Err()
}

// SessionStats returns statistics for a session.
func (p *PostgresConversation) SessionStats(ctx context.Context, sessionID string) (*SessionStats, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as message_count,
			MIN(created_at) as first_message,
			MAX(created_at) as last_message
		FROM %s
		WHERE session_id = $1
	`, p.table)

	var stats SessionStats
	stats.SessionID = sessionID

	var firstMsg, lastMsg sql.NullTime
	err := p.db.QueryRowContext(ctx, query, sessionID).Scan(
		&stats.MessageCount,
		&firstMsg,
		&lastMsg,
	)
	if err != nil {
		return nil, err
	}

	if firstMsg.Valid {
		stats.FirstMessage = firstMsg.Time
	}
	if lastMsg.Valid {
		stats.LastMessage = lastMsg.Time
	}

	return &stats, nil
}

// SessionStats contains statistics about a conversation session.
type SessionStats struct {
	SessionID    string
	MessageCount int
	FirstMessage time.Time
	LastMessage  time.Time
}

func (p *PostgresConversation) queryMessages(ctx context.Context, query string, args ...any) ([]ConversationMessage, error) {
	rows, err := p.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ConversationMessage
	for rows.Next() {
		var msg ConversationMessage
		var toolCallID sql.NullString
		var metadataJSON []byte

		err := rows.Scan(
			&msg.ID,
			&msg.SessionID,
			&msg.Role,
			&msg.Content,
			&toolCallID,
			&metadataJSON,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if toolCallID.Valid {
			msg.ToolCallID = toolCallID.String
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &msg.Metadata); err != nil {
				// Log but don't fail on metadata parse error
				msg.Metadata = nil
			}
		}

		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// Close closes the database connection.
func (p *PostgresConversation) Close() error {
	return p.db.Close()
}
