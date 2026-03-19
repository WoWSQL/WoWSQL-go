package WOWSQL

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FilterOperator represents a query filter operator.
type FilterOperator string

const (
	OpEq         FilterOperator = "eq"
	OpNeq        FilterOperator = "neq"
	OpGt         FilterOperator = "gt"
	OpGte        FilterOperator = "gte"
	OpLt         FilterOperator = "lt"
	OpLte        FilterOperator = "lte"
	OpLike       FilterOperator = "like"
	OpIsNull     FilterOperator = "is"
	OpIsNotNull  FilterOperator = "is_not"
	OpIn         FilterOperator = "in"
	OpNotIn      FilterOperator = "not_in"
	OpBetween    FilterOperator = "between"
	OpNotBetween FilterOperator = "not_between"
)

// SortDirection represents sort direction.
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// FilterExpression represents a filter condition.
type FilterExpression struct {
	Column    string         `json:"column"`
	Operator  FilterOperator `json:"operator"`
	Value     interface{}    `json:"value,omitempty"`
	LogicalOp string        `json:"logical_op,omitempty"`
}

// QueryBuilder provides a fluent interface for building queries.
type QueryBuilder struct {
	client         *Client
	tableName      string
	columns        []string
	filters        []FilterExpression
	groupByColumns []string
	havingFilters  []HavingFilter
	orderColumn    string
	orderDirection SortDirection
	limitValue     *int
	offsetValue    *int
}

// Select specifies columns to select.
func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

// Filter adds a filter condition.
func (qb *QueryBuilder) Filter(column string, operator FilterOperator, value interface{}, logicalOp ...string) *QueryBuilder {
	logicalOperator := "AND"
	if len(logicalOp) > 0 && logicalOp[0] != "" {
		logicalOperator = logicalOp[0]
	}
	qb.filters = append(qb.filters, FilterExpression{
		Column:    column,
		Operator:  operator,
		Value:     value,
		LogicalOp: logicalOperator,
	})
	return qb
}

// Eq adds an equality filter.
func (qb *QueryBuilder) Eq(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpEq, value)
}

// Neq adds a not-equal filter.
func (qb *QueryBuilder) Neq(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpNeq, value)
}

// Gt adds a greater-than filter.
func (qb *QueryBuilder) Gt(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpGt, value)
}

// Gte adds a greater-than-or-equal filter.
func (qb *QueryBuilder) Gte(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpGte, value)
}

// Lt adds a less-than filter.
func (qb *QueryBuilder) Lt(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpLt, value)
}

// Lte adds a less-than-or-equal filter.
func (qb *QueryBuilder) Lte(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpLte, value)
}

// Like adds a LIKE pattern filter.
func (qb *QueryBuilder) Like(column string, pattern string) *QueryBuilder {
	return qb.Filter(column, OpLike, pattern)
}

// IsNull adds an IS NULL filter.
func (qb *QueryBuilder) IsNull(column string) *QueryBuilder {
	return qb.Filter(column, OpIsNull, nil)
}

// IsNotNull adds an IS NOT NULL filter.
func (qb *QueryBuilder) IsNotNull(column string) *QueryBuilder {
	return qb.Filter(column, OpIsNotNull, nil)
}

// In adds an IN filter.
func (qb *QueryBuilder) In(column string, values []interface{}) *QueryBuilder {
	return qb.Filter(column, OpIn, values)
}

// NotIn adds a NOT IN filter.
func (qb *QueryBuilder) NotIn(column string, values []interface{}) *QueryBuilder {
	return qb.Filter(column, OpNotIn, values)
}

// Between adds a BETWEEN filter.
func (qb *QueryBuilder) Between(column string, minValue, maxValue interface{}) *QueryBuilder {
	return qb.Filter(column, OpBetween, []interface{}{minValue, maxValue})
}

// NotBetween adds a NOT BETWEEN filter.
func (qb *QueryBuilder) NotBetween(column string, minValue, maxValue interface{}) *QueryBuilder {
	return qb.Filter(column, OpNotBetween, []interface{}{minValue, maxValue})
}

// Or adds an OR filter condition.
func (qb *QueryBuilder) Or(column string, operator FilterOperator, value interface{}) *QueryBuilder {
	return qb.Filter(column, operator, value, "OR")
}

// GroupBy sets columns to group by.
func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.groupByColumns = columns
	return qb
}

// Having adds a HAVING clause filter.
func (qb *QueryBuilder) Having(column string, operator string, value interface{}) *QueryBuilder {
	qb.havingFilters = append(qb.havingFilters, HavingFilter{
		Column:   column,
		Operator: operator,
		Value:    value,
	})
	return qb
}

// OrderBy sets the order column and direction.
func (qb *QueryBuilder) OrderBy(column string, direction SortDirection) *QueryBuilder {
	qb.orderColumn = column
	qb.orderDirection = direction
	return qb
}

// Order is an alias for OrderBy.
func (qb *QueryBuilder) Order(column string, direction SortDirection) *QueryBuilder {
	return qb.OrderBy(column, direction)
}

// Limit sets the maximum number of results.
func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitValue = &limit
	return qb
}

// Offset sets the number of records to skip.
func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetValue = &offset
	return qb
}

// Execute executes the query and returns results.
// Uses POST /{table}/query for advanced queries, GET /{table} for simple ones.
func (qb *QueryBuilder) Execute() (*QueryResponse, error) {
	body := qb.buildQueryBody()

	if qb.hasAdvancedFeatures(body) {
		return qb.executePost(body)
	}
	return qb.executeGet(body)
}

// Get is an alias for Execute.
func (qb *QueryBuilder) Get() (*QueryResponse, error) {
	return qb.Execute()
}

// First retrieves only the first result.
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

// Single retrieves exactly one result. Returns an error if zero or more than
// one record is found.
func (qb *QueryBuilder) Single() (map[string]interface{}, error) {
	qb.Limit(2)
	result, err := qb.Execute()
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, &WOWSQLError{Message: "No records found"}
	}
	if len(result.Data) > 1 {
		return nil, &WOWSQLError{Message: "Multiple records found, expected exactly one"}
	}
	return result.Data[0], nil
}

// Count returns the total number of records matching the current filters.
func (qb *QueryBuilder) Count() (int, error) {
	savedColumns := qb.columns
	savedGroupBy := qb.groupByColumns
	savedHaving := qb.havingFilters
	savedOrder := qb.orderColumn
	savedDir := qb.orderDirection

	qb.columns = []string{"COUNT(*) as count"}
	qb.groupByColumns = nil
	qb.havingFilters = nil
	qb.orderColumn = ""
	qb.orderDirection = ""

	result, err := qb.Execute()

	qb.columns = savedColumns
	qb.groupByColumns = savedGroupBy
	qb.havingFilters = savedHaving
	qb.orderColumn = savedOrder
	qb.orderDirection = savedDir

	if err != nil {
		return 0, err
	}
	if len(result.Data) > 0 {
		if v, ok := result.Data[0]["count"]; ok {
			switch n := v.(type) {
			case float64:
				return int(n), nil
			case json.Number:
				i, _ := n.Int64()
				return int(i), nil
			}
		}
	}
	return 0, nil
}

// Paginate returns a paginated result set.
func (qb *QueryBuilder) Paginate(page, perPage int) (*PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	offsetVal := (page - 1) * perPage
	qb.Limit(perPage).Offset(offsetVal)

	result, err := qb.Execute()
	if err != nil {
		return nil, err
	}

	total := result.Count
	if result.Total != nil {
		total = *result.Total
	}
	totalPages := 0
	if total > 0 {
		totalPages = (total + perPage - 1) / perPage
	}

	return &PaginatedResponse{
		Data:       result.Data,
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// Update updates records matching the query.
func (qb *QueryBuilder) Update(data map[string]interface{}) (*UpdateResponse, error) {
	body := map[string]interface{}{
		"data": data,
	}
	if len(qb.filters) > 0 {
		body["filters"] = qb.filters
	}

	resp, err := qb.client.doRequest("PUT", fmt.Sprintf("/%s", qb.tableName), body)
	if err != nil {
		return nil, err
	}

	var result UpdateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// Delete deletes records matching the query.
func (qb *QueryBuilder) Delete() (*DeleteResponse, error) {
	body := make(map[string]interface{})
	if len(qb.filters) > 0 {
		body["filters"] = qb.filters
	}

	resp, err := qb.client.doRequest("DELETE", fmt.Sprintf("/%s", qb.tableName), body)
	if err != nil {
		return nil, err
	}

	var result DeleteResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

// ── Internal helpers ────────────────────────────────────────────

func (qb *QueryBuilder) hasAdvancedFeatures(body map[string]interface{}) bool {
	if _, ok := body["group_by"]; ok {
		return true
	}
	if _, ok := body["having"]; ok {
		return true
	}
	if filters, ok := body["filters"].([]FilterExpression); ok {
		for _, f := range filters {
			switch f.Operator {
			case OpIn, OpNotIn, OpBetween, OpNotBetween:
				return true
			}
		}
	}
	return false
}

func (qb *QueryBuilder) executePost(body map[string]interface{}) (*QueryResponse, error) {
	postBody := make(map[string]interface{})

	if cols, ok := body["select"]; ok {
		postBody["select"] = cols
	}
	if filters := body["filters"]; filters != nil {
		postBody["filters"] = filters
	}
	if gb, ok := body["group_by"]; ok {
		postBody["group_by"] = gb
	}
	if h, ok := body["having"]; ok {
		postBody["having"] = h
	}
	if ob, ok := body["order_by"]; ok {
		postBody["order_by"] = ob
		if od, ok := body["order_direction"]; ok {
			postBody["order_direction"] = od
		}
	}
	if lim, ok := body["limit"]; ok {
		postBody["limit"] = lim
	}
	if off, ok := body["offset"]; ok {
		postBody["offset"] = off
	}

	resp, err := qb.client.doRequest("POST", fmt.Sprintf("/%s/query", qb.tableName), postBody)
	if err != nil {
		return nil, err
	}

	var result QueryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

func (qb *QueryBuilder) executeGet(body map[string]interface{}) (*QueryResponse, error) {
	params := make([]string, 0)

	if cols, ok := body["select"].([]string); ok {
		params = append(params, "select="+strings.Join(cols, ","))
	}

	if filters, ok := body["filters"].([]FilterExpression); ok && len(filters) > 0 {
		parts := make([]string, 0, len(filters))
		for _, f := range filters {
			if _, isList := f.Value.([]interface{}); isList {
				return qb.executePost(body)
			}
			parts = append(parts, fmt.Sprintf("%s.%s.%v", f.Column, f.Operator, f.Value))
		}
		params = append(params, "filter="+strings.Join(parts, ","))
	}

	if ob, ok := body["order_by"].(string); ok && ob != "" {
		params = append(params, "order="+ob)
		if od, ok := body["order_direction"].(SortDirection); ok {
			params = append(params, "order_direction="+string(od))
		}
	}
	if lim, ok := body["limit"]; ok {
		params = append(params, fmt.Sprintf("limit=%v", lim))
	}
	if off, ok := body["offset"]; ok {
		params = append(params, fmt.Sprintf("offset=%v", off))
	}

	path := fmt.Sprintf("/%s", qb.tableName)
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}

	resp, err := qb.client.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var result QueryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &result, nil
}

func (qb *QueryBuilder) buildQueryBody() map[string]interface{} {
	body := make(map[string]interface{})

	if len(qb.columns) > 0 {
		body["select"] = qb.columns
	}
	if len(qb.filters) > 0 {
		body["filters"] = qb.filters
	}
	if len(qb.groupByColumns) > 0 {
		body["group_by"] = qb.groupByColumns
	}
	if len(qb.havingFilters) > 0 {
		body["having"] = qb.havingFilters
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
