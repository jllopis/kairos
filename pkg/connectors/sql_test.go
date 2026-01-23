// Copyright 2026 © The Kairos Authors
// SPDX-License-Identifier: Apache-2.0

package connectors

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite" // Pure Go SQLite driver
)

// TestSQLConnectorFromTables tests creating a connector from pre-defined tables.
func TestSQLConnectorFromTables(t *testing.T) {
	tables := map[string]*SQLTable{
		"users": {
			Name:       "users",
			PrimaryKey: []string{"id"},
			Columns: []SQLColumn{
				{Name: "id", Type: "INTEGER", IsPrimary: true, HasDefault: true},
				{Name: "name", Type: "VARCHAR", Nullable: false},
				{Name: "email", Type: "VARCHAR", Nullable: false},
				{Name: "age", Type: "INTEGER", Nullable: true},
			},
		},
	}

	c := NewSQLConnectorFromTables(tables)

	if len(c.Tables()) != 1 {
		t.Errorf("Expected 1 table, got %d", len(c.Tables()))
	}
}

// TestSQLToolGeneration tests tool generation from tables.
func TestSQLToolGeneration(t *testing.T) {
	tables := map[string]*SQLTable{
		"users": {
			Name:       "users",
			PrimaryKey: []string{"id"},
			Columns: []SQLColumn{
				{Name: "id", Type: "INTEGER", IsPrimary: true, HasDefault: true},
				{Name: "name", Type: "VARCHAR", Nullable: false},
				{Name: "email", Type: "VARCHAR", Nullable: false},
			},
		},
	}

	c := NewSQLConnectorFromTables(tables)
	tools := c.Tools()

	// Should generate 5 tools per table: list, get, create, update, delete
	if len(tools) != 5 {
		t.Errorf("Expected 5 tools, got %d", len(tools))
	}

	// Check tool names
	expectedNames := map[string]bool{
		"list_users":   true,
		"get_users":    true,
		"create_users": true,
		"update_users": true,
		"delete_users": true,
	}

	for _, tool := range tools {
		name := tool.ToolDefinition().Function.Name
		if !expectedNames[name] {
			t.Errorf("Unexpected tool name: %s", name)
		}
	}
}

// TestSQLReadOnlyMode tests read-only mode.
func TestSQLReadOnlyMode(t *testing.T) {
	tables := map[string]*SQLTable{
		"users": {
			Name:       "users",
			PrimaryKey: []string{"id"},
			Columns: []SQLColumn{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
				{Name: "name", Type: "VARCHAR"},
			},
		},
	}

	c := NewSQLConnectorFromTables(tables, WithSQLReadOnly())
	tools := c.Tools()

	// Should only generate list and get (read-only)
	if len(tools) != 2 {
		t.Errorf("Expected 2 tools in read-only mode, got %d", len(tools))
	}

	for _, tool := range tools {
		name := tool.ToolDefinition().Function.Name
		if name != "list_users" && name != "get_users" {
			t.Errorf("Unexpected write tool in read-only mode: %s", name)
		}
	}
}

// TestSQLToolPrefix tests tool name prefixing.
func TestSQLToolPrefix(t *testing.T) {
	tables := map[string]*SQLTable{
		"users": {
			Name:       "users",
			PrimaryKey: []string{"id"},
			Columns: []SQLColumn{
				{Name: "id", Type: "INTEGER", IsPrimary: true},
			},
		},
	}

	c := NewSQLConnectorFromTables(tables, WithSQLToolPrefix("db"))
	tools := c.Tools()

	for _, tool := range tools {
		name := tool.ToolDefinition().Function.Name
		if name[:3] != "db_" {
			t.Errorf("Expected tool name to start with 'db_', got %s", name)
		}
	}
}

// TestSQLColumnToJSONSchema tests SQL type to JSON Schema mapping.
func TestSQLColumnToJSONSchema(t *testing.T) {
	c := NewSQLConnectorFromTables(nil)

	tests := []struct {
		column   SQLColumn
		expected string
	}{
		{SQLColumn{Name: "id", Type: "INTEGER"}, "integer"},
		{SQLColumn{Name: "id", Type: "BIGINT"}, "integer"},
		{SQLColumn{Name: "id", Type: "SMALLINT"}, "integer"},
		{SQLColumn{Name: "price", Type: "FLOAT"}, "number"},
		{SQLColumn{Name: "price", Type: "DOUBLE"}, "number"},
		{SQLColumn{Name: "price", Type: "DECIMAL(10,2)"}, "number"},
		{SQLColumn{Name: "price", Type: "NUMERIC"}, "number"},
		{SQLColumn{Name: "active", Type: "BOOLEAN"}, "boolean"},
		{SQLColumn{Name: "active", Type: "BOOL"}, "boolean"},
		{SQLColumn{Name: "name", Type: "VARCHAR(255)"}, "string"},
		{SQLColumn{Name: "name", Type: "TEXT"}, "string"},
		{SQLColumn{Name: "created", Type: "DATETIME"}, "string"},
		{SQLColumn{Name: "created", Type: "TIMESTAMP"}, "string"},
		{SQLColumn{Name: "data", Type: "JSON"}, "object"},
	}

	for _, tt := range tests {
		schema := c.columnToJSONSchema(tt.column)
		if schema["type"] != tt.expected {
			t.Errorf("For %s (%s), expected type %s, got %s",
				tt.column.Name, tt.column.Type, tt.expected, schema["type"])
		}
	}
}

// TestSQLWithSQLite tests actual SQL operations with SQLite.
func TestSQLWithSQLite(t *testing.T) {
	// Create in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			email TEXT NOT NULL,
			age INTEGER
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Create connector
	c, err := NewSQLConnector(db, "sqlite")
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	// Check introspection
	if len(c.Tables()) != 1 {
		t.Errorf("Expected 1 table, got %d", len(c.Tables()))
	}

	table := c.Tables()["users"]
	if table == nil {
		t.Fatal("Expected users table")
	}

	if len(table.Columns) != 4 {
		t.Errorf("Expected 4 columns, got %d", len(table.Columns))
	}

	// Test tools
	tools := c.Tools()
	if len(tools) != 5 {
		t.Errorf("Expected 5 tools, got %d", len(tools))
	}

	ctx := context.Background()

	// Test create
	result, err := c.Execute(ctx, "create_users", map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   float64(30),
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	createResult := result.(map[string]interface{})
	if createResult["rows_affected"].(int64) != 1 {
		t.Errorf("Expected 1 row affected")
	}

	// Test list
	result, err = c.Execute(ctx, "list_users", map[string]interface{}{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	listResult := result.([]map[string]interface{})
	if len(listResult) != 1 {
		t.Errorf("Expected 1 record, got %d", len(listResult))
	}

	if listResult[0]["name"] != "John Doe" {
		t.Errorf("Expected name 'John Doe', got %v", listResult[0]["name"])
	}

	// Test get
	result, err = c.Execute(ctx, "get_users", map[string]interface{}{
		"id": int64(1),
	})
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	getResult := result.(map[string]interface{})
	if getResult["email"] != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got %v", getResult["email"])
	}

	// Test update
	result, err = c.Execute(ctx, "update_users", map[string]interface{}{
		"id":   int64(1),
		"name": "Jane Doe",
	})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	updateResult := result.(map[string]interface{})
	if updateResult["rows_affected"].(int64) != 1 {
		t.Errorf("Expected 1 row affected")
	}

	// Verify update
	result, err = c.Execute(ctx, "get_users", map[string]interface{}{"id": int64(1)})
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if result.(map[string]interface{})["name"] != "Jane Doe" {
		t.Errorf("Update not applied")
	}

	// Test delete
	result, err = c.Execute(ctx, "delete_users", map[string]interface{}{
		"id": int64(1),
	})
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	deleteResult := result.(map[string]interface{})
	if deleteResult["rows_affected"].(int64) != 1 {
		t.Errorf("Expected 1 row affected")
	}

	// Verify delete
	result, err = c.Execute(ctx, "list_users", map[string]interface{}{})
	if err != nil {
		t.Fatalf("List after delete failed: %v", err)
	}
	if len(result.([]map[string]interface{})) != 0 {
		t.Errorf("Expected 0 records after delete")
	}
}

// TestSQLListWithFilters tests list with filter conditions.
func TestSQLListWithFilters(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE products (
			id INTEGER PRIMARY KEY,
			name TEXT,
			category TEXT,
			price REAL
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Insert test data
	db.Exec(`INSERT INTO products (name, category, price) VALUES ('Apple', 'Fruit', 1.50)`)
	db.Exec(`INSERT INTO products (name, category, price) VALUES ('Banana', 'Fruit', 0.75)`)
	db.Exec(`INSERT INTO products (name, category, price) VALUES ('Carrot', 'Vegetable', 1.00)`)

	c, err := NewSQLConnector(db, "sqlite")
	if err != nil {
		t.Fatalf("Failed to create connector: %v", err)
	}

	ctx := context.Background()

	// Test filter by category
	result, err := c.Execute(ctx, "list_products", map[string]interface{}{
		"filters": map[string]interface{}{
			"category": "Fruit",
		},
	})
	if err != nil {
		t.Fatalf("List with filter failed: %v", err)
	}

	listResult := result.([]map[string]interface{})
	if len(listResult) != 2 {
		t.Errorf("Expected 2 fruits, got %d", len(listResult))
	}

	// Test limit
	result, err = c.Execute(ctx, "list_products", map[string]interface{}{
		"limit": float64(1),
	})
	if err != nil {
		t.Fatalf("List with limit failed: %v", err)
	}

	listResult = result.([]map[string]interface{})
	if len(listResult) != 1 {
		t.Errorf("Expected 1 result with limit, got %d", len(listResult))
	}
}
