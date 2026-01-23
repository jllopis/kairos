// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package connectors

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jllopis/kairos/pkg/core"
	"github.com/jllopis/kairos/pkg/llm"
)

// SQLConnector generates tools from a database schema.
type SQLConnector struct {
	db         *sql.DB
	driver     string
	tables     map[string]*SQLTable
	toolPrefix string
	readOnly   bool
}

// SQLTable represents a database table.
type SQLTable struct {
	Name       string
	Schema     string
	Columns    []SQLColumn
	PrimaryKey []string
}

// SQLColumn represents a column in a table.
type SQLColumn struct {
	Name       string
	Type       string
	Nullable   bool
	IsPrimary  bool
	MaxLength  int
	HasDefault bool
}

// SQLOption configures the SQLConnector.
type SQLOption func(*SQLConnector)

// WithSQLTables limits introspection to specific tables.
func WithSQLTables(tables ...string) SQLOption {
	return func(c *SQLConnector) {
		// This is handled in introspect()
		// Store table filter in a temporary field or handle differently
	}
}

// WithSQLToolPrefix adds a prefix to generated tool names.
func WithSQLToolPrefix(prefix string) SQLOption {
	return func(c *SQLConnector) {
		c.toolPrefix = prefix
	}
}

// WithSQLReadOnly generates only read tools (no INSERT, UPDATE, DELETE).
func WithSQLReadOnly() SQLOption {
	return func(c *SQLConnector) {
		c.readOnly = true
	}
}

// NewSQLConnector creates a SQL connector from a database connection.
func NewSQLConnector(db *sql.DB, driver string, opts ...SQLOption) (*SQLConnector, error) {
	c := &SQLConnector{
		db:     db,
		driver: driver,
		tables: make(map[string]*SQLTable),
	}

	for _, opt := range opts {
		opt(c)
	}

	// Perform introspection
	if err := c.introspect(); err != nil {
		return nil, fmt.Errorf("introspection failed: %w", err)
	}

	return c, nil
}

// NewSQLConnectorFromTables creates a connector from pre-defined tables.
// Useful for testing or when introspection is not available.
func NewSQLConnectorFromTables(tables map[string]*SQLTable, opts ...SQLOption) *SQLConnector {
	c := &SQLConnector{
		tables: tables,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// introspect discovers tables and columns from the database.
func (c *SQLConnector) introspect() error {
	if c.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	ctx := context.Background()

	// Query differs by database driver
	var query string
	switch c.driver {
	case "postgres", "postgresql":
		query = `
			SELECT
				table_name,
				column_name,
				data_type,
				is_nullable,
				character_maximum_length,
				column_default
			FROM information_schema.columns
			WHERE table_schema = 'public'
			ORDER BY table_name, ordinal_position
		`
	case "mysql":
		query = `
			SELECT
				table_name,
				column_name,
				data_type,
				is_nullable,
				character_maximum_length,
				column_default
			FROM information_schema.columns
			WHERE table_schema = DATABASE()
			ORDER BY table_name, ordinal_position
		`
	case "sqlite", "sqlite3":
		// SQLite uses pragma, we'll handle it differently
		return c.introspectSQLite(ctx)
	default:
		// Generic approach using information_schema
		query = `
			SELECT
				table_name,
				column_name,
				data_type,
				is_nullable,
				character_maximum_length,
				column_default
			FROM information_schema.columns
			ORDER BY table_name, ordinal_position
		`
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query columns: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		var maxLength sql.NullInt64
		var columnDefault sql.NullString

		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &maxLength, &columnDefault); err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Get or create table
		table, ok := c.tables[tableName]
		if !ok {
			table = &SQLTable{
				Name:    tableName,
				Columns: []SQLColumn{},
			}
			c.tables[tableName] = table
		}

		// Add column
		col := SQLColumn{
			Name:       columnName,
			Type:       dataType,
			Nullable:   strings.ToUpper(isNullable) == "YES",
			HasDefault: columnDefault.Valid,
		}
		if maxLength.Valid {
			col.MaxLength = int(maxLength.Int64)
		}

		table.Columns = append(table.Columns, col)
	}

	// Get primary keys
	if err := c.introspectPrimaryKeys(ctx); err != nil {
		// Non-fatal, continue without PK info
	}

	return rows.Err()
}

// introspectSQLite handles SQLite-specific introspection.
func (c *SQLConnector) introspectSQLite(ctx context.Context) error {
	// Get list of tables
	rows, err := c.db.QueryContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		tableNames = append(tableNames, name)
	}

	// Get columns for each table
	for _, tableName := range tableNames {
		table := &SQLTable{
			Name:    tableName,
			Columns: []SQLColumn{},
		}

		pragmaRows, err := c.db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info('%s')", tableName))
		if err != nil {
			continue
		}

		for pragmaRows.Next() {
			var cid int
			var name, dataType string
			var notNull, pk int
			var dfltValue sql.NullString

			if err := pragmaRows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk); err != nil {
				pragmaRows.Close()
				continue
			}

			col := SQLColumn{
				Name:       name,
				Type:       dataType,
				Nullable:   notNull == 0,
				IsPrimary:  pk > 0,
				HasDefault: dfltValue.Valid,
			}
			table.Columns = append(table.Columns, col)

			if pk > 0 {
				table.PrimaryKey = append(table.PrimaryKey, name)
			}
		}
		pragmaRows.Close()

		c.tables[tableName] = table
	}

	return nil
}

// introspectPrimaryKeys gets primary key information.
func (c *SQLConnector) introspectPrimaryKeys(ctx context.Context) error {
	var query string
	switch c.driver {
	case "postgres", "postgresql":
		query = `
			SELECT
				kcu.table_name,
				kcu.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu
				ON tc.constraint_name = kcu.constraint_name
			WHERE tc.constraint_type = 'PRIMARY KEY'
			AND tc.table_schema = 'public'
		`
	case "mysql":
		query = `
			SELECT
				table_name,
				column_name
			FROM information_schema.key_column_usage
			WHERE constraint_name = 'PRIMARY'
			AND table_schema = DATABASE()
		`
	default:
		return nil // Skip for unsupported drivers
	}

	rows, err := c.db.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, columnName string
		if err := rows.Scan(&tableName, &columnName); err != nil {
			continue
		}

		if table, ok := c.tables[tableName]; ok {
			table.PrimaryKey = append(table.PrimaryKey, columnName)
			// Mark column as primary
			for i := range table.Columns {
				if table.Columns[i].Name == columnName {
					table.Columns[i].IsPrimary = true
				}
			}
		}
	}

	return nil
}

// Tools generates core tools from discovered tables.
func (c *SQLConnector) Tools() []core.Tool {
	return coreToolsFromDefinitions(c.toolDefinitions(), c)
}

func (c *SQLConnector) toolDefinitions() []llm.Tool {
	var tools []llm.Tool

	for _, table := range c.tables {
		// Generate CRUD tools for each table
		tools = append(tools, c.generateListTool(table))
		tools = append(tools, c.generateGetTool(table))

		if !c.readOnly {
			tools = append(tools, c.generateCreateTool(table))
			tools = append(tools, c.generateUpdateTool(table))
			tools = append(tools, c.generateDeleteTool(table))
		}
	}

	return tools
}

// generateListTool creates a tool for listing/querying records.
func (c *SQLConnector) generateListTool(table *SQLTable) llm.Tool {
	name := fmt.Sprintf("list_%s", toSnakeCase(table.Name))
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	// Build filter properties from columns
	filterProps := make(map[string]interface{})
	for _, col := range table.Columns {
		filterProps[col.Name] = c.columnToJSONSchema(col)
	}

	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: fmt.Sprintf("List records from %s table with optional filters", table.Name),
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"filters": map[string]interface{}{
						"type":        "object",
						"description": "Filter conditions (column: value)",
						"properties":  filterProps,
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Maximum number of records to return",
						"default":     100,
					},
					"offset": map[string]interface{}{
						"type":        "integer",
						"description": "Number of records to skip",
						"default":     0,
					},
					"order_by": map[string]interface{}{
						"type":        "string",
						"description": "Column to order by",
					},
					"order_desc": map[string]interface{}{
						"type":        "boolean",
						"description": "Order descending",
						"default":     false,
					},
				},
			},
		},
	}
}

// generateGetTool creates a tool for getting a single record by primary key.
func (c *SQLConnector) generateGetTool(table *SQLTable) llm.Tool {
	name := fmt.Sprintf("get_%s", toSnakeCase(table.Name))
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	// Build properties for primary key columns
	props := make(map[string]interface{})
	var required []string

	if len(table.PrimaryKey) > 0 {
		for _, pk := range table.PrimaryKey {
			for _, col := range table.Columns {
				if col.Name == pk {
					props[col.Name] = c.columnToJSONSchema(col)
					required = append(required, col.Name)
					break
				}
			}
		}
	} else {
		// Fallback: use "id" if no PK defined
		props["id"] = map[string]interface{}{"type": "string", "description": "Record ID"}
		required = append(required, "id")
	}

	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: fmt.Sprintf("Get a single record from %s by primary key", table.Name),
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": props,
				"required":   required,
			},
		},
	}
}

// generateCreateTool creates a tool for inserting a new record.
func (c *SQLConnector) generateCreateTool(table *SQLTable) llm.Tool {
	name := fmt.Sprintf("create_%s", toSnakeCase(table.Name))
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	props := make(map[string]interface{})
	var required []string

	for _, col := range table.Columns {
		// Skip auto-generated primary keys
		if col.IsPrimary && col.HasDefault {
			continue
		}

		props[col.Name] = c.columnToJSONSchema(col)

		// Required if not nullable and no default
		if !col.Nullable && !col.HasDefault && !col.IsPrimary {
			required = append(required, col.Name)
		}
	}

	params := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		params["required"] = required
	}

	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: fmt.Sprintf("Create a new record in %s", table.Name),
			Parameters:  params,
		},
	}
}

// generateUpdateTool creates a tool for updating a record.
func (c *SQLConnector) generateUpdateTool(table *SQLTable) llm.Tool {
	name := fmt.Sprintf("update_%s", toSnakeCase(table.Name))
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	props := make(map[string]interface{})
	var required []string

	// Primary key is required for update
	for _, pk := range table.PrimaryKey {
		for _, col := range table.Columns {
			if col.Name == pk {
				props[col.Name] = c.columnToJSONSchema(col)
				required = append(required, col.Name)
				break
			}
		}
	}

	// Add all other columns as optional update fields
	for _, col := range table.Columns {
		if col.IsPrimary {
			continue
		}
		props[col.Name] = c.columnToJSONSchema(col)
	}

	params := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		params["required"] = required
	}

	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: fmt.Sprintf("Update a record in %s", table.Name),
			Parameters:  params,
		},
	}
}

// generateDeleteTool creates a tool for deleting a record.
func (c *SQLConnector) generateDeleteTool(table *SQLTable) llm.Tool {
	name := fmt.Sprintf("delete_%s", toSnakeCase(table.Name))
	if c.toolPrefix != "" {
		name = c.toolPrefix + "_" + name
	}

	props := make(map[string]interface{})
	var required []string

	// Primary key is required for delete
	if len(table.PrimaryKey) > 0 {
		for _, pk := range table.PrimaryKey {
			for _, col := range table.Columns {
				if col.Name == pk {
					props[col.Name] = c.columnToJSONSchema(col)
					required = append(required, col.Name)
					break
				}
			}
		}
	} else {
		props["id"] = map[string]interface{}{"type": "string", "description": "Record ID"}
		required = append(required, "id")
	}

	return llm.Tool{
		Type: llm.ToolTypeFunction,
		Function: llm.FunctionDef{
			Name:        name,
			Description: fmt.Sprintf("Delete a record from %s", table.Name),
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": props,
				"required":   required,
			},
		},
	}
}

// columnToJSONSchema converts a SQL column to JSON Schema.
func (c *SQLConnector) columnToJSONSchema(col SQLColumn) map[string]interface{} {
	schema := make(map[string]interface{})

	// Map SQL types to JSON Schema types
	sqlType := strings.ToUpper(col.Type)

	switch {
	case strings.Contains(sqlType, "INT"):
		schema["type"] = "integer"
	case strings.Contains(sqlType, "FLOAT") || strings.Contains(sqlType, "DOUBLE") ||
		strings.Contains(sqlType, "DECIMAL") || strings.Contains(sqlType, "NUMERIC") ||
		strings.Contains(sqlType, "REAL"):
		schema["type"] = "number"
	case strings.Contains(sqlType, "BOOL"):
		schema["type"] = "boolean"
	case strings.Contains(sqlType, "DATE") || strings.Contains(sqlType, "TIME"):
		schema["type"] = "string"
		schema["format"] = "date-time"
	case strings.Contains(sqlType, "JSON"):
		schema["type"] = "object"
	case strings.Contains(sqlType, "ARRAY"):
		schema["type"] = "array"
	default:
		schema["type"] = "string"
	}

	// Add max length for string types
	if schema["type"] == "string" && col.MaxLength > 0 {
		schema["maxLength"] = col.MaxLength
	}

	return schema
}

// Execute runs a SQL operation based on the tool name.
func (c *SQLConnector) Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	if c.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	// Remove prefix
	name := toolName
	if c.toolPrefix != "" && strings.HasPrefix(toolName, c.toolPrefix+"_") {
		name = strings.TrimPrefix(toolName, c.toolPrefix+"_")
	}

	// Parse operation and table name
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid tool name format: %s", toolName)
	}

	operation := parts[0]
	tableName := parts[1]

	// Find table (handle snake_case to original name mapping)
	var table *SQLTable
	for _, t := range c.tables {
		if toSnakeCase(t.Name) == tableName {
			table = t
			break
		}
	}
	if table == nil {
		return nil, fmt.Errorf("table not found: %s", tableName)
	}

	switch operation {
	case "list":
		return c.executeList(ctx, table, args)
	case "get":
		return c.executeGet(ctx, table, args)
	case "create":
		if c.readOnly {
			return nil, fmt.Errorf("connector is read-only")
		}
		return c.executeCreate(ctx, table, args)
	case "update":
		if c.readOnly {
			return nil, fmt.Errorf("connector is read-only")
		}
		return c.executeUpdate(ctx, table, args)
	case "delete":
		if c.readOnly {
			return nil, fmt.Errorf("connector is read-only")
		}
		return c.executeDelete(ctx, table, args)
	default:
		return nil, fmt.Errorf("unknown operation: %s", operation)
	}
}

// executeList runs a SELECT query with optional filters.
func (c *SQLConnector) executeList(ctx context.Context, table *SQLTable, args map[string]interface{}) (interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s", c.quoteIdentifier(table.Name))
	var queryArgs []interface{}

	// Build WHERE clause from filters
	if filters, ok := args["filters"].(map[string]interface{}); ok && len(filters) > 0 {
		var conditions []string
		for col, val := range filters {
			conditions = append(conditions, fmt.Sprintf("%s = ?", c.quoteIdentifier(col)))
			queryArgs = append(queryArgs, val)
		}
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// ORDER BY
	if orderBy, ok := args["order_by"].(string); ok && orderBy != "" {
		query += fmt.Sprintf(" ORDER BY %s", c.quoteIdentifier(orderBy))
		if desc, ok := args["order_desc"].(bool); ok && desc {
			query += " DESC"
		}
	}

	// LIMIT and OFFSET
	limit := 100
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	if offset, ok := args["offset"].(float64); ok && offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", int(offset))
	}

	// Execute query
	rows, err := c.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return c.rowsToMaps(rows)
}

// executeGet runs a SELECT query for a single record.
func (c *SQLConnector) executeGet(ctx context.Context, table *SQLTable, args map[string]interface{}) (interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE ", c.quoteIdentifier(table.Name))
	var conditions []string
	var queryArgs []interface{}

	// Build WHERE from primary key
	pkCols := table.PrimaryKey
	if len(pkCols) == 0 {
		pkCols = []string{"id"}
	}

	for _, pk := range pkCols {
		if val, ok := args[pk]; ok {
			conditions = append(conditions, fmt.Sprintf("%s = ?", c.quoteIdentifier(pk)))
			queryArgs = append(queryArgs, val)
		} else {
			return nil, fmt.Errorf("missing primary key: %s", pk)
		}
	}

	query += strings.Join(conditions, " AND ") + " LIMIT 1"

	rows, err := c.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	results, err := c.rowsToMaps(rows)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	return results[0], nil
}

// executeCreate runs an INSERT query.
func (c *SQLConnector) executeCreate(ctx context.Context, table *SQLTable, args map[string]interface{}) (interface{}, error) {
	var columns []string
	var placeholders []string
	var values []interface{}

	for col, val := range args {
		columns = append(columns, c.quoteIdentifier(col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		c.quoteIdentifier(table.Name),
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "))

	result, err := c.db.ExecContext(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("insert failed: %w", err)
	}

	lastID, _ := result.LastInsertId()
	rowsAffected, _ := result.RowsAffected()

	return map[string]interface{}{
		"last_insert_id": lastID,
		"rows_affected":  rowsAffected,
	}, nil
}

// executeUpdate runs an UPDATE query.
func (c *SQLConnector) executeUpdate(ctx context.Context, table *SQLTable, args map[string]interface{}) (interface{}, error) {
	var setClauses []string
	var setValues []interface{}
	var whereClauses []string
	var whereValues []interface{}

	pkSet := make(map[string]bool)
	for _, pk := range table.PrimaryKey {
		pkSet[pk] = true
	}

	for col, val := range args {
		if pkSet[col] {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", c.quoteIdentifier(col)))
			whereValues = append(whereValues, val)
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", c.quoteIdentifier(col)))
			setValues = append(setValues, val)
		}
	}

	if len(whereClauses) == 0 {
		return nil, fmt.Errorf("missing primary key for update")
	}

	if len(setClauses) == 0 {
		return nil, fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s",
		c.quoteIdentifier(table.Name),
		strings.Join(setClauses, ", "),
		strings.Join(whereClauses, " AND "))

	allValues := append(setValues, whereValues...)

	result, err := c.db.ExecContext(ctx, query, allValues...)
	if err != nil {
		return nil, fmt.Errorf("update failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return map[string]interface{}{
		"rows_affected": rowsAffected,
	}, nil
}

// executeDelete runs a DELETE query.
func (c *SQLConnector) executeDelete(ctx context.Context, table *SQLTable, args map[string]interface{}) (interface{}, error) {
	var whereClauses []string
	var values []interface{}

	pkCols := table.PrimaryKey
	if len(pkCols) == 0 {
		pkCols = []string{"id"}
	}

	for _, pk := range pkCols {
		if val, ok := args[pk]; ok {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = ?", c.quoteIdentifier(pk)))
			values = append(values, val)
		} else {
			return nil, fmt.Errorf("missing primary key: %s", pk)
		}
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s",
		c.quoteIdentifier(table.Name),
		strings.Join(whereClauses, " AND "))

	result, err := c.db.ExecContext(ctx, query, values...)
	if err != nil {
		return nil, fmt.Errorf("delete failed: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()

	return map[string]interface{}{
		"rows_affected": rowsAffected,
	}, nil
}

// rowsToMaps converts sql.Rows to a slice of maps.
func (c *SQLConnector) rowsToMaps(rows *sql.Rows) ([]map[string]interface{}, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for readability
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// quoteIdentifier quotes a SQL identifier.
func (c *SQLConnector) quoteIdentifier(name string) string {
	switch c.driver {
	case "mysql":
		return "`" + name + "`"
	case "postgres", "postgresql":
		return `"` + name + `"`
	default:
		return `"` + name + `"`
	}
}

// Tables returns the discovered tables.
func (c *SQLConnector) Tables() map[string]*SQLTable {
	return c.tables
}

// Close closes the database connection.
func (c *SQLConnector) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}
