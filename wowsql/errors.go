package WOWSQL

import (
	"encoding/json"
	"fmt"
)

// WOWSQLError represents a base WOWSQL error
type WOWSQLError struct {
	Message    string
	StatusCode int
	Response   map[string]interface{}
}

func (e *WOWSQLError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("WOWSQLError(%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("WOWSQLError: %s", e.Message)
}

// AuthenticationError represents authentication errors
type AuthenticationError struct {
	WOWSQLError
}

// NotFoundError represents not found errors
type NotFoundError struct {
	WOWSQLError
}

// RateLimitError represents rate limit errors
type RateLimitError struct {
	WOWSQLError
}

// NetworkError represents network errors
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("NetworkError: %s", e.Err.Error())
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

// StorageError represents storage errors
type StorageError struct {
	Message    string
	StatusCode int
	Response   map[string]interface{}
	Err        error
}

func (e *StorageError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("StorageError(%d): %s", e.StatusCode, e.Message)
	}
	if e.Err != nil {
		return fmt.Sprintf("StorageError: %s", e.Err.Error())
	}
	return fmt.Sprintf("StorageError: %s", e.Message)
}

func (e *StorageError) Unwrap() error {
	return e.Err
}

// StorageLimitExceededError represents storage limit exceeded errors
type StorageLimitExceededError struct {
	Message        string
	RequiredBytes  int64
	AvailableBytes int64
	StatusCode     int
	Response       map[string]interface{}
}

func (e *StorageLimitExceededError) Error() string {
	if e.RequiredBytes > 0 && e.AvailableBytes > 0 {
		return fmt.Sprintf("StorageLimitExceededError: %s (Required: %s, Available: %s)",
			e.Message,
			formatBytes(e.RequiredBytes),
			formatBytes(e.AvailableBytes))
	}
	return fmt.Sprintf("StorageLimitExceededError: %s", e.Message)
}

// parseError parses an error response
func parseError(statusCode int, body []byte) error {
	var errorResponse map[string]interface{}
	_ = json.Unmarshal(body, &errorResponse)

	message := "Request failed"
	if msg, ok := errorResponse["error"].(string); ok {
		message = msg
	} else if msg, ok := errorResponse["message"].(string); ok {
		message = msg
	} else if msg, ok := errorResponse["detail"].(string); ok {
		message = msg
	}

	if message == "Request failed" {
		message = fmt.Sprintf("Request failed with status %d", statusCode)
	}

	switch statusCode {
	case 401, 403:
		return &AuthenticationError{
			WOWSQLError: WOWSQLError{
				Message:    message,
				StatusCode: statusCode,
				Response:   errorResponse,
			},
		}
	case 404:
		return &NotFoundError{
			WOWSQLError: WOWSQLError{
				Message:    message,
				StatusCode: statusCode,
				Response:   errorResponse,
			},
		}
	case 429:
		return &RateLimitError{
			WOWSQLError: WOWSQLError{
				Message:    message,
				StatusCode: statusCode,
				Response:   errorResponse,
			},
		}
	default:
		return &WOWSQLError{
			Message:    message,
			StatusCode: statusCode,
			Response:   errorResponse,
		}
	}
}

// parseStorageError parses a storage error response
func parseStorageError(statusCode int, body []byte) error {
	var errorResponse map[string]interface{}
	_ = json.Unmarshal(body, &errorResponse)

	message := "Request failed"
	if msg, ok := errorResponse["error"].(string); ok {
		message = msg
	} else if msg, ok := errorResponse["message"].(string); ok {
		message = msg
	} else if msg, ok := errorResponse["detail"].(string); ok {
		message = msg
	}

	if message == "Request failed" {
		message = fmt.Sprintf("Request failed with status %d", statusCode)
	}

	if statusCode == 413 {
		return &StorageLimitExceededError{
			Message:    message,
			StatusCode: statusCode,
			Response:   errorResponse,
		}
	}

	return &StorageError{
		Message:    message,
		StatusCode: statusCode,
		Response:   errorResponse,
	}
}

