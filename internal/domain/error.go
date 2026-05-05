package domain

import (
	"fmt"
	"net/http"
)

// AppError adalah error standar seluruh aplikasi
type AppError struct {
	Code       string
	Message    string
	HTTPStatus int
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Err }

func NewNotFoundError(resource, id string) *AppError {
	return &AppError{
		Code:       resource + "_NOT_FOUND",
		Message:    fmt.Sprintf("%s with id %s not found", resource, id),
		HTTPStatus: http.StatusNotFound,
	}
}

func NewValidationError(message string) *AppError {
	return &AppError{
		Code:       "VALIDATION_ERROR",
		Message:    message,
		HTTPStatus: http.StatusUnprocessableEntity,
	}
}

func NewInternalError(err error) *AppError {
	return &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "an internal error occurred",
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

// Sentinel errors
var (
	ErrTransactionNotFound = &AppError{Code: "TRANSACTION_NOT_FOUND", HTTPStatus: http.StatusNotFound}
	ErrInvalidSignature    = &AppError{Code: "INVALID_SIGNATURE", HTTPStatus: http.StatusUnauthorized}
	ErrDuplicateRequest    = &AppError{Code: "DUPLICATE_REQUEST", HTTPStatus: http.StatusConflict}
	ErrInvalidStatus       = &AppError{Code: "INVALID_STATUS", HTTPStatus: http.StatusUnprocessableEntity}
)
