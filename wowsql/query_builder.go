package WOWSQL

import (
	"encoding/json"
	"fmt"
	"net/url"
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
	OpILike      FilterOperator = "ilike"
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
	LogicalOp string         `json:"logical_op,omitempty"`
}

// orderItem holds a single column order specification.
type orderItem struct {
	column    string
	direction SortDirection
}

// QueryBuilder provides a fluent interface for building PostgREST queries.
type QueryBuilder struct {
	client         *Client
	tableName      string
	columns        []string
	filters        []FilterExpression
	groupByColumns []string
	havingFilters  []HavingFilter
	orderItems     []orderItem
	limitValue     *int
	offsetValue    *int
}

// filtersToParams translates FilterExpression slice to PostgREST query parameters.
func filtersToParams(filters []FilterExpression) url.Values {
	params := url.Values{}
	for _, f := range filters {
		col := f.Column
		op := f.Operator
		val := f.Value

		switch op {
		case OpEq:
			params.Set(col, fmt.Sprintf("eq.%v", val))
		case OpNeq:
			params.Set(col, fmt.Sprintf("neq.%v", val))
		case OpGt:
			params.Set(col, fmt.Sprintf("gt.%v", val))
		case OpGte:
			params.Set(col, fmt.Sprintf("gte.%v", val))
		case OpLt:
			params.Set(col, fmt.Sprintf("lt.%v", val))
		case OpLte:
			params.Set(col, fmt.Sprintf("lte.%v", val))
		case OpLike:
			pattern := strings.ReplaceAll(fmt.Sprintf("%v", val), "%", "*")
			params.Set(col, "like."+pattern)
		case OpILike:
			pattern := strings.ReplaceAll(fmt.Sprintf("%v", val), "%", "*")
			params.Set(col, "ilike."+pattern)
		case OpIsNull:
			if val == nil {
				params.Set(col, "is.null")
			} else {
				params.Set(col, fmt.Sprintf("is.%v", val))
			}
		case OpIsNotNull:
			if val == nil {
				params.Set(col, "not.is.null")
			} else {
				params.Set(col, fmt.Sprintf("not.is.%v", val))
			}
		case OpIn:
			if vals, ok := val.([]interface{}); ok {
				parts := make([]string, len(vals))
				for i, v := range vals {
					parts[i] = fmt.Sprintf("%v", v)
				}
				params.Set(col, "in.("+strings.Join(parts, ",")+")")
			}
		case OpNotIn:
			if vals, ok := val.([]interface{}); ok {
				parts := make([]string, len(vals))
				for i, v := range vals {
					parts[i] = fmt.Sprintf("%v", v)
				}
				params.Set(col, "not.in.("+strings.Join(parts, ",")+")")
			}
		case OpBetween:
			if vals, ok := val.([]interface{}); ok && len(vals) == 2 {
				params.Set(col, fmt.Sprintf("gte.%v", vals[0]))
				params.Set(col+"_lte", fmt.Sprintf("lte.%v", vals[1]))
			}
		case OpNotBetween:
			if vals, ok := val.([]interface{}); ok && len(vals) == 2 {
				params.Set(col+"_lt", fmt.Sprintf("lt.%v", vals[0]))
				params.Set(col+"_gt", fmt.Sprintf("gt.%v", vals[1]))
			}
		}
	}
	return params
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

func (qb *QueryBuilder) Eq(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpEq, value)
}

func (qb *QueryBuilder) Neq(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpNeq, value)
}

func (qb *QueryBuilder) Gt(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpGt, value)
}

func (qb *QueryBuilder) Gte(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpGte, value)
}

func (qb *QueryBuilder) Lt(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpLt, value)
}

func (qb *QueryBuilder) Lte(column string, value interface{}) *QueryBuilder {
	return qb.Filter(column, OpLte, value)
}

func (qb *QueryBuilder) Like(column string, pattern string) *QueryBuilder {
	return qb.Filter(column, OpLike, pattern)
}

func (qb *QueryBuilder) ILike(column string, pattern string) *QueryBuilder {
	return qb.Filter(column, OpILike, pattern)
}

func (qb *QueryBuilder) IsNull(column string) *QueryBuilder {
	return qb.Filter(column, OpIsNull, nil)
}

func (qb *QueryBuilder) IsNotNull(column string) *QueryBuilder {
	return qb.Filter(column, OpIsNotNull, nil)
}

func (qb *QueryBuilder) In(column string, values []interface{}) *QueryBuilder {
	return qb.Filter(column, OpIn, values)
}

func (qb *QueryBuilder) NotIn(column string, values []interface{}) *QueryBuilder {
	return qb.Filter(column, OpNotIn, values)
}

func (qb *QueryBuilder) Between(column string, minValue, maxValue interface{}) *QueryBuilder {
	return qb.Filter(column, OpBetween, []interface{}{minValue, maxValue})
}

func (qb *QueryBuilder) NotBetween(column string, minValue, maxValue interface{}) *QueryBuilder {
	return qb.Filter(column, OpNotBetween, []interface{}{minValue, maxValue})
}

func (qb *QueryBuilder) Or(column string, operator FilterOperator, value interface{}) *QueryBuilder {
	return qb.Filter(column, operator, value, "OR")
}

func (qb *QueryBuilder) GroupBy(columns ...string) *QueryBuilder {
	qb.groupByColumns = columns
	return qb
}

func (qb *QueryBuilder) Having(column string, operator string, value interface{}) *QueryBuilder {
	qb.havingFilters = append(qb.havingFilters, HavingFilter{
		Column:   column,
		Operator: operator,
		Value:    value,
	})
	return qb
}

// OrderBy sets a column to order by.
func (qb *QueryBuilder) OrderBy(column string, direction SortDirection) *QueryBuilder {
	qb.orderItems = append(qb.orderItems, orderItem{column: column, direction: direction})
	return qb
}

// Order is an alias for OrderBy.
func (qb *QueryBuilder) Order(column string, direction SortDirection) *QueryBuilder {
	return qb.OrderBy(column, direction)
}

func (qb *QueryBuilder) Limit(limit int) *QueryBuilder {
	qb.limitValue = &limit
	return qb
}

func (qb *QueryBuilder) Offset(offset int) *QueryBuilder {
	qb.offsetValue = &offset
	return qb
}

// Execute runs the query against PostgREST using native query parameters.
func (qb *QueryBuilder) Execute() (*QueryResponse, error) {
	return qb.Get()
}

// Get runs the query and returns matching records.
func (qb *QueryBuilder) Get() (*QueryResponse, error) {
	params := filtersToParams(qb.filters)

	// SELECT
	if len(qb.columns) > 0 {
		sel := strings.Join(qb.columns, ",")
		// Merge group-by columns into select
		if len(qb.groupByColumns) > 0 {
			existing := strings.Split(sel, ",")
			merged := make([]string, 0, len(existing)+len(qb.groupByColumns))
			seen := map[string]bool{}
			for _, c := range existing {
				seen[c] = true
				merged = append(merged, c)
			}
			for _, c := range qb.groupByColumns {
				if !seen[c] {
					merged = append(merged, c)
				}
			}
			sel = strings.Join(merged, ",")
		}
		params.Set("select", sel)
	} else if len(qb.groupByColumns) > 0 {
		params.Set("select", strings.Join(qb.groupByColumns, ","))
	}

	// ORDER — PostgREST: ?order=col.asc,col2.desc
	if len(qb.orderItems) > 0 {
		parts := make([]string, len(qb.orderItems))
		for i, o := range qb.orderItems {
			dir := o.direction
			if dir == "" {
				dir = SortAsc
			}
			parts[i] = fmt.Sprintf("%s.%s", o.column, dir)
		}
		params.Set("order", strings.Join(parts, ","))
	}

	// LIMIT / OFFSET
	if qb.limitValue != nil {
		params.Set("limit", fmt.Sprintf("%d", *qb.limitValue))
	}
	if qb.offsetValue != nil {
		params.Set("offset", fmt.Sprintf("%d", *qb.offsetValue))
	}

	path := fmt.Sprintf("/%s", qb.tableName)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, resp, err := qb.client.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	var rawData []map[string]interface{}
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	total := len(rawData)
	if resp != nil {
		total = parseTotalFromContentRange(resp.Header.Get("Content-Range"), total)
	}

	limit := 100
	offset := 0
	if qb.limitValue != nil {
		limit = *qb.limitValue
	}
	if qb.offsetValue != nil {
		offset = *qb.offsetValue
	}

	return &QueryResponse{
		Data:   rawData,
		Count:  len(rawData),
		Total:  &total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// First retrieves the first matching record.
func (qb *QueryBuilder) First() (map[string]interface{}, error) {
	qb.Limit(1)
	result, err := qb.Get()
	if err != nil {
		return nil, err
	}
	if len(result.Data) == 0 {
		return nil, nil
	}
	return result.Data[0], nil
}

// Single retrieves exactly one record. Returns an error if zero or more than one found.
func (qb *QueryBuilder) Single() (map[string]interface{}, error) {
	qb.Limit(2)
	result, err := qb.Get()
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
	savedCols := qb.columns
	savedLimit := qb.limitValue
	savedOffset := qb.offsetValue
	savedOrder := qb.orderItems

	zero := 0
	qb.columns = nil
	qb.limitValue = &zero
	qb.offsetValue = nil
	qb.orderItems = nil

	result, err := qb.Get()

	qb.columns = savedCols
	qb.limitValue = savedLimit
	qb.offsetValue = savedOffset
	qb.orderItems = savedOrder

	if err != nil {
		return 0, err
	}
	if result.Total != nil {
		return *result.Total, nil
	}
	return result.Count, nil
}

// Sum returns the sum of a column for matching records.
func (qb *QueryBuilder) Sum(column string) (float64, error) {
	savedCols := qb.columns
	savedLimit := qb.limitValue
	savedOffset := qb.offsetValue

	qb.columns = []string{fmt.Sprintf("sum(%s)", column)}
	qb.limitValue = nil
	qb.offsetValue = nil

	result, err := qb.Get()

	qb.columns = savedCols
	qb.limitValue = savedLimit
	qb.offsetValue = savedOffset

	if err != nil {
		return 0, err
	}
	if len(result.Data) > 0 {
		if v, ok := result.Data[0]["sum"]; ok {
			return toFloat64(v), nil
		}
	}
	return 0, nil
}

// Avg returns the average of a column for matching records.
func (qb *QueryBuilder) Avg(column string) (float64, error) {
	savedCols := qb.columns
	savedLimit := qb.limitValue
	savedOffset := qb.offsetValue

	qb.columns = []string{fmt.Sprintf("avg(%s)", column)}
	qb.limitValue = nil
	qb.offsetValue = nil

	result, err := qb.Get()

	qb.columns = savedCols
	qb.limitValue = savedLimit
	qb.offsetValue = savedOffset

	if err != nil {
		return 0, err
	}
	if len(result.Data) > 0 {
		if v, ok := result.Data[0]["avg"]; ok {
			return toFloat64(v), nil
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

	result, err := qb.Get()
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

// Update updates records matching the query filters.
func (qb *QueryBuilder) Update(data map[string]interface{}) (*UpdateResponse, error) {
	params := filtersToParams(qb.filters)
	path := fmt.Sprintf("/%s", qb.tableName)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, _, err := qb.client.doRequestWithHeaders("PATCH", path, data,
		map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		var single map[string]interface{}
		if err2 := json.Unmarshal(body, &single); err2 != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
	}
	return &UpdateResponse{
		Message:      "Records updated successfully",
		AffectedRows: len(rows),
	}, nil
}

// Delete deletes records matching the query filters.
func (qb *QueryBuilder) Delete() (*DeleteResponse, error) {
	params := filtersToParams(qb.filters)
	path := fmt.Sprintf("/%s", qb.tableName)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	body, _, err := qb.client.doRequestWithHeaders("DELETE", path, nil,
		map[string]string{"Prefer": "return=representation"})
	if err != nil {
		return nil, err
	}

	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		rows = nil
	}
	return &DeleteResponse{
		Message:      "Records deleted successfully",
		AffectedRows: len(rows),
	}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func parseTotalFromContentRange(header string, fallback int) int {
	if header == "" {
		return fallback
	}
	parts := strings.Split(header, "/")
	if len(parts) < 2 {
		return fallback
	}
	var n int
	if _, err := fmt.Sscanf(parts[1], "%d", &n); err != nil {
		return fallback
	}
	return n
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case json.Number:
		f, _ := n.Float64()
		return f
	case string:
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	}
	return 0
}
