package WOWSQL

import (
	"encoding/json"
	"fmt"
)

// FilterOperator represents a query filter operator
type FilterOperator string

const (
	OpEq     FilterOperator = "eq"
	OpNeq    FilterOperator = "neq"
	OpGt     FilterOperator = "gt"
	OpGte    FilterOperator = "gte"
	OpLt     FilterOperator = "lt"
	OpLte    FilterOperator = "lte"
	OpLike   FilterOperator = "like"
	OpIsNull FilterOperator = "is"
)

// SortDirection represents sort direction
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// FilterExpression represents a filter condition
type FilterExpression struct {
	Column   string         `json:"column"`
	Operator FilterOperator `json:"operator"`
	Value    interface{}    `json:"value,omitempty"`
}

// QueryBuilder provides a fluent interface for building queries
type QueryBuilder struct {
	client         *Client
	tableName      string
	columns        []string
	filters        []FilterExpression
	orderColumn    string
	orderDirection SortDirection
	limitValue     *int
	offsetValue    *int
}

// Select specifies columns to select
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Eq adds an equality filter
func (qb *QueryBuilder) Eq(column string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpEq,
		Value:    value,
	})
	return qb
}

// Neq adds a not-equal filter
func (qb *QueryBuilder) Neq(column string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpNeq,
		Value:    value,
	})
	return qb
}

// Gt adds a greater-than filter
func (qb *QueryBuilder) Gt(column string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpGt,
		Value:    value,
	})
	return qb
}

// Gte adds a greater-than-or-equal filter
func (qb *QueryBuilder) Gte(column string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpGte,
		Value:    value,
	})
	return qb
}

// Lt adds a less-than filter
func (qb *QueryBuilder) Lt(column string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpLt,
		Value:    value,
	})
	return qb
}

// Lte adds a less-than-or-equal filter
func (qb *QueryBuilder) Lte(column string, value interface{}) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpLte,
		Value:    value,
	})
	return qb
}

// Like adds a LIKE pattern filter
func (qb *QueryBuilder) Like(column string, pattern string) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpLike,
		Value:    pattern,
	})
	return qb
}

// IsNull adds an IS NULL filter
func (qb *QueryBuilder) IsNull(column string) *QueryBuilder {
	qb.filters = append(qb.filters, FilterExpression{
		Column:   column,
		Operator: OpIsNull,
	})
	return qb
}

// OrderBy sets the order column and direction
func (qb *QueryBuilder) OrderBy(column string, direction SortDirection) *QueryBuilder {
	qb.orderColumn = column
	qb.orderDirection = direction
	return qb
}

// Limit sets the limit
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitValue = &limit
	return qb
}

// Offset sets the offset
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetValue = &offset
	return qb
}

// Execute executes the query and returns results
func (qb *QueryBuilder) Execute() (*QueryResponse, error) {
	body := qb.buildQueryBody()

	resp, err := qb.client.doRequest("POST", fmt.Sprintf("/api/v1/tables/%s/query", qb.tableName), body)
	if err != nil {
		return nil, err
	}

	var result QueryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Get is an alias for Execute
func (qb *QueryBuilder) Get() (*QueryResponse, error) {
	return qb.Execute()
}

// First retrieves only the first result
func (qb *QueryBuilder) First() (map[string]interface{}, error) {
	qb.Limit(1)
	result, err := qb.Execute()
	if err != nil {
		return nil, err
	}

	if len(result.Data) == 0 {
		return nil, nil
	}

	return result.Data[0], nil
}

// Update updates records matching the query
func (qb *QueryBuilder) Update(data map[string]interface{}) (*UpdateResponse, error) {
	body := map[string]interface{}{
		"data": data,
	}

	if len(qb.filters) > 0 {
		body["filters"] = qb.filters
	}

	resp, err := qb.client.doRequest("PUT", fmt.Sprintf("/api/v1/tables/%s", qb.tableName), body)
	if err != nil {
		return nil, err
	}

	var result UpdateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Delete deletes records matching the query
func (qb *QueryBuilder) Delete() (*DeleteResponse, error) {
	body := make(map[string]interface{})

	if len(qb.filters) > 0 {
		body["filters"] = qb.filters
	}

	resp, err := qb.client.doRequest("DELETE", fmt.Sprintf("/api/v1/tables/%s", qb.tableName), body)
	if err != nil {
		return nil, err
	}

	var result DeleteResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// buildQueryBody builds the query request body
func (qb *QueryBuilder) buildQueryBody() map[string]interface{} {
	body := make(map[string]interface{})

	if len(qb.columns) > 0 {
		body["columns"] = qb.columns
	}

	if len(qb.filters) > 0 {
		body["filters"] = qb.filters
	}

	if qb.orderColumn != "" {
		body["order_by"] = qb.orderColumn
		body["order_direction"] = qb.orderDirection
	}

	if qb.limitValue != nil {
		body["limit"] = *qb.limitValue
	}

	if qb.offsetValue != nil {
		body["offset"] = *qb.offsetValue
	}

	return body
}

