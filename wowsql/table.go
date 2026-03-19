package WOWSQL

import (
	"encoding/json"
	"fmt"
)

// Table represents a database table with fluent API.
type Table struct {
	client    *Client
	tableName string
}

// ── Query builders ──────────────────────────────────────────────

// Select creates a new QueryBuilder with column selection.
func (t *Table) Select(columns ...string) *QueryBuilder {
	return (&QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		filters:   make([]FilterExpression, 0),
	}).Select(columns...)
}

// Filter starts a query with a filter.
func (t *Table) Filter(column string, operator FilterOperator, value interface{}, logicalOp ...string) *QueryBuilder {
	return (&QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		filters:   make([]FilterExpression, 0),
	}).Filter(column, operator, value, logicalOp...)
}

// Get retrieves all records (shorthand for Select("*").Get()).
func (t *Table) Get() (*QueryResponse, error) {
	return t.Select("*").Get()
}

// GetByID retrieves a single record by ID.
func (t *Table) GetByID(id interface{}) (map[string]interface{}, error) {
	resp, err := t.client.doRequest("GET", fmt.Sprintf("/%s/%v", t.tableName, id), nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return result, nil
}

// ── Write operations ────────────────────────────────────────────

// Create inserts a new record.
func (t *Table) Create(data map[string]interface{}) (*CreateResponse, error) {
	resp, err := t.client.doRequest("POST", fmt.Sprintf("/%s", t.tableName), data)
	if err != nil {
		return nil, err
	}

	var result CreateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// Insert is an alias for Create.
func (t *Table) Insert(data map[string]interface{}) (*CreateResponse, error) {
	return t.Create(data)
}

// BulkInsert inserts multiple records. Attempts a single batch POST first;
// falls back to individual inserts if the server does not support batch creation.
func (t *Table) BulkInsert(records []map[string]interface{}) ([]*CreateResponse, error) {
	if len(records) == 0 {
		return []*CreateResponse{}, nil
	}

	resp, err := t.client.doRequest("POST", fmt.Sprintf("/%s", t.tableName), records)
	if err == nil {
		// Try to parse as array
		var results []*CreateResponse
		if json.Unmarshal(resp, &results) == nil {
			return results, nil
		}
		// Try single result
		var single CreateResponse
		if json.Unmarshal(resp, &single) == nil {
			return []*CreateResponse{&single}, nil
		}
	}

	// Fallback to individual inserts
	results := make([]*CreateResponse, 0, len(records))
	for _, record := range records {
		r, insertErr := t.Create(record)
		if insertErr != nil {
			return results, insertErr
		}
		results = append(results, r)
	}
	return results, nil
}

// Upsert inserts or updates based on a conflict column.
func (t *Table) Upsert(data map[string]interface{}, onConflict string) (map[string]interface{}, error) {
	if onConflict == "" {
		onConflict = "id"
	}

	conflictValue, ok := data[onConflict]
	if !ok || conflictValue == nil {
		resp, err := t.Create(data)
		if err != nil {
			return nil, err
		}
		b, _ := json.Marshal(resp)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		return m, nil
	}

	existing, err := (&QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		filters:   make([]FilterExpression, 0),
	}).Eq(onConflict, conflictValue).First()
	if err != nil {
		return nil, err
	}

	if existing != nil {
		updateData := make(map[string]interface{})
		for k, v := range data {
			if k != onConflict {
				updateData[k] = v
			}
		}
		if len(updateData) == 0 {
			return map[string]interface{}{"message": "No changes", "affected_rows": 0}, nil
		}
		resp, err := t.Update(conflictValue, updateData)
		if err != nil {
			return nil, err
		}
		b, _ := json.Marshal(resp)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		return m, nil
	}

	resp, err := t.Create(data)
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(resp)
	var m map[string]interface{}
	_ = json.Unmarshal(b, &m)
	return m, nil
}

// Update updates a record by ID.
func (t *Table) Update(recordID interface{}, data map[string]interface{}) (*UpdateResponse, error) {
	resp, err := t.client.doRequest("PATCH", fmt.Sprintf("/%s/%v", t.tableName, recordID), data)
	if err != nil {
		return nil, err
	}

	var result UpdateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// Delete deletes a record by ID.
func (t *Table) Delete(recordID interface{}) (*DeleteResponse, error) {
	resp, err := t.client.doRequest("DELETE", fmt.Sprintf("/%s/%v", t.tableName, recordID), nil)
	if err != nil {
		return nil, err
	}

	var result DeleteResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// ── Convenience shortcuts ───────────────────────────────────────

func (t *Table) newBuilder() *QueryBuilder {
	return &QueryBuilder{
		client:    t.client,
		tableName: t.tableName,
		filters:   make([]FilterExpression, 0),
	}
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

// Paginate paginates all records in this table.
func (t *Table) Paginate(page, perPage int) (*PaginatedResponse, error) {
	return t.newBuilder().Paginate(page, perPage)
}

// Where creates a new QueryBuilder for filtered operations (legacy helper).
func (t *Table) Where() *QueryBuilder {
	return t.newBuilder()
}
