package response

import "time"

type ApiResponse[T any] struct {
	Success   bool         `json:"success"`
	Code      string       `json:"code"`
	Message   string       `json:"message"`
	Data      T            `json:"data,omitempty"`
	Error     *ErrorDetail `json:"error,omitempty"`
	Meta      *Meta        `json:"meta,omitempty"` // for paginated response
	RequestID string       `json:"request_id"`
	Timestamp time.Time    `json:"timestamp"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"` // validation error per field
}

type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}
