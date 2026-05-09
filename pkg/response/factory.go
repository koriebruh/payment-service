package response

import (
	"time"

	"github.com/koriebruh/payment-service/internal/domain"
)

type ApiResponseFactory struct{}

func NewApiResponseFactory() *ApiResponseFactory {
	return &ApiResponseFactory{}
}

// Success response dengan data
func (f *ApiResponseFactory) Success(
	requestID string,
	code string,
	message string,
	data any,
) ApiResponse[any] {
	return ApiResponse[any]{
		Success:   true,
		Code:      code,
		Message:   message,
		Data:      data,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

// Success response tanpa data
func (f *ApiResponseFactory) SuccessNoData(
	requestID string,
	code string,
	message string,
) ApiResponse[any] {
	return ApiResponse[any]{
		Success:   true,
		Code:      code,
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

// Error response
func (f *ApiResponseFactory) Error(
	requestID string,
	appErr *domain.AppError,
) ApiResponse[any] {
	if appErr == nil {
		appErr = domain.NewInternalError(nil)
	}

	return ApiResponse[any]{
		Success: false,
		Code:    appErr.Code,
		Message: appErr.Message,
		Error: &ErrorDetail{
			Code:    appErr.Code,
			Message: appErr.Message,
		},
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}

// Validation error response
func (f *ApiResponseFactory) ValidationError(
	requestID string,
	fields map[string]string,
) ApiResponse[any] {
	return ApiResponse[any]{
		Success: false,
		Code:    "VALIDATION_ERROR",
		Message: "request validation failed",
		Error: &ErrorDetail{
			Code:    "VALIDATION_ERROR",
			Message: "one or more fields are invalid",
			Fields:  fields,
		},
		RequestID: requestID,
		Timestamp: time.Now().UTC(),
	}
}
