package WOWSQL

import (
	"encoding/json"
	"fmt"
)

// Table represents a database table with a fluent query API.
// All operations communicate directly with PostgREST (/rest/v1).
type Table struct {
	client    *Client
	tableName string
}

// ── Query chain entry points ─────────────────────────────────────────────────

func (t *Table) newBuilder() *QueryBuilder {
	return &QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		filters:   make([]FilterExpression, 0),
	}
}

// Select creates a QueryBuilder with column selection.
func (t *Table) Select(columns ...string) *QueryBuilder {
	return t.newBuilder().Select(columns...)
}

// Filter starts a query with a filter.
func (t *Table) Filter(column string, operator FilterOperator, value interface{}, logicalOp ...string) *QueryBuilder {
	return t.newBuilder().Filter(column, operator, value, logicalOp...)
}

// Get retrieves all records.
func (t *Table) Get() (*QueryResponse, error) {
	return t.newBuilder().Get()
}

// Eq starts a query filtering where column equals value.
func (t *Table) Eq(column string, value interface{}) *QueryBuilder {
	return t.newBuilder().Eq(column, value)
}

// Neq starts a query filtering where column != value.
func (t *Table) Neq(column string, value interface{}) *QueryBuilder {
	return t.newBuilder().Neq(column, value)
}

// Gt starts a query filtering where column > value.
func (t *Table) Gt(column string, value interface{}) *QueryBuilder {
	return t.newBuilder().Gt(column, value)
}

// Gte starts a query filtering where column >= value.
func (t *Table) Gte(column string, value interface{}) *QueryBuilder {
	return t.newBuilder().Gte(column, value)
}

// Lt starts a query filtering where column < value.
func (t *Table) Lt(column string, value interface{}) *QueryBuilder {
	return t.newBuilder().Lt(column, value)
}

// Lte starts a query filtering where column <= value.
func (t *Table) Lte(column string, value interface{}) *QueryBuilder {
	return t.newBuilder().Lte(column, value)
}

// OrderBy starts a query with ordering.
func (t *Table) OrderBy(column string, direction SortDirection) *QueryBuilder {
	return t.newBuilder().OrderBy(column, direction)
}

// Count returns the total record count for this table.
func (t *Table) Count() (int, error) {
	return t.newBuilder().Count()
}

// Paginate paginates all records.
func (t *Table) Paginate(page, perPage int) (*PaginatedResponse, error) {
	return t.newBuilder().Paginate(page, perPage)
}

// ── Single-record CRUD ───────────────────────────────────────────────────────

// GetByID retrieves a single record by primary-key value using PostgREST id=eq.{id}.
func (t *Table) GetByID(id interface{}) (map[string]interface{}, error) {
	path := fmt.Sprintf("/%s?id=eq.%v", t.tableName, id)
	body, _, err := t.client.doRequestWithHeaders("GET", path, nil,
		map[string]string{"Accept": "application/vnd.pgrst.object+json"})
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

// Create inserts a new record and returns the created row.
func (t *Table) Create(data map[string]interface{}) (*CreateResponse, error) {
	body, _, err := t.client.doRequestWithHeaders("POST", "/"+t.tableName, data,
		map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
		var single map[string]interface{}
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return &CreateResponse{ID: single["id"], Message: "Record created successfully"}, nil
	}
	return &CreateResponse{ID: rows[0]["id"], Message: "Record created successfully"}, nil
}

// Insert is an alias for Create.
func (t *Table) Insert(data map[string]interface{}) (*CreateResponse, error) {
	return t.Create(data)
}

// BulkInsert inserts multiple records in a single request.
func (t *Table) BulkInsert(records []map[string]interface{}) ([]*CreateResponse, error) {
	if len(records) == 0 {
		return []*CreateResponse{}, nil
	}
	body, _, err := t.client.doRequestWithHeaders("POST", "/"+t.tableName, records,
		map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	results := make([]*CreateResponse, len(rows))
	for i, row := range rows {
		results[i] = &CreateResponse{ID: row["id"], Message: "Record created successfully"}
	}
	return results, nil
}

// Upsert inserts or updates based on a conflict column using PostgREST merge-duplicates.
func (t *Table) Upsert(data map[string]interface{}, onConflict string) (*CreateResponse, error) {
	if onConflict == "" {
		onConflict = "id"
	}
	body, _, err := t.client.doRequestWithHeaders("POST", "/"+t.tableName, data,
		map[string]string{
			"Prefer":      "return=representation,resolution=merge-duplicates",
			"on-conflict": onConflict,
		})
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil || len(rows) == 0 {
		var single map[string]interface{}
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		return &CreateResponse{ID: single["id"], Message: "Record upserted successfully"}, nil
	}
	return &CreateResponse{ID: rows[0]["id"], Message: "Record upserted successfully"}, nil
}

// Update updates a record by primary-key value using PostgREST id=eq.{id}.
func (t *Table) Update(recordID interface{}, data map[string]interface{}) (*UpdateResponse, error) {
	path := fmt.Sprintf("/%s?id=eq.%v", t.tableName, recordID)
	body, _, err := t.client.doRequestWithHeaders("PATCH", path, data,
		map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	_ = json.Unmarshal(body, &rows)
	return &UpdateResponse{
		Message:      "Record updated successfully",
		AffectedRows: len(rows),
	}, nil
}

// Delete deletes a record by primary-key value using PostgREST id=eq.{id}.
func (t *Table) Delete(recordID interface{}) (*DeleteResponse, error) {
	path := fmt.Sprintf("/%s?id=eq.%v", t.tableName, recordID)
	body, _, err := t.client.doRequestWithHeaders("DELETE", path, nil,
		map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return nil, err
	}
	var rows []map[string]interface{}
	_ = json.Unmarshal(body, &rows)
	return &DeleteResponse{
		Message:      "Record deleted successfully",
		AffectedRows: len(rows),
	}, nil
}
