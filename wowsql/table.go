package WOWSQL

import (
	"encoding/json"
	"fmt"
)

// Table represents a database table with fluent API
type Table struct {
	client    *Client
	tableName string
}

// Select creates a new QueryBuilder for select queries
func (t *Table) Select(columns ...string) *QueryBuilder {
	return &QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		columns:   columns,
		filters:   make([]FilterExpression, 0),
	}
}

// Get retrieves all records (shorthand for Select("*"))
func (t *Table) Get() *QueryBuilder {
	return t.Select("*")
}

// GetByID retrieves a single record by ID
func (t *Table) GetByID(id interface{}) *QueryBuilder {
	return t.Select("*").Eq("id", id).Limit(1)
}

// Insert inserts a new record
func (t *Table) Insert(data map[string]interface{}) (*CreateResponse, error) {
	resp, err := t.client.doRequest("POST", fmt.Sprintf("/api/v1/tables/%s", t.tableName), data)
	if err != nil {
		return nil, err
	}

	var result CreateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// UpdateByID updates a record by ID
func (t *Table) UpdateByID(id interface{}, data map[string]interface{}) (*UpdateResponse, error) {
	return t.Where().Eq("id", id).Update(data)
}

// DeleteByID deletes a record by ID
func (t *Table) DeleteByID(id interface{}) (*DeleteResponse, error) {
	return t.Where().Eq("id", id).Delete()
}

// Where creates a new QueryBuilder for filtered operations
func (t *Table) Where() *QueryBuilder {
	return &QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		filters:   make([]FilterExpression, 0),
	}
}

