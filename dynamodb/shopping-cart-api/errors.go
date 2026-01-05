package main

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/gin-gonic/gin"
)

// ErrorCode represents standardized error codes
type ErrorCode string

const (
	ErrCodeNotFound           ErrorCode = "CART_NOT_FOUND"
	ErrCodeBadRequest         ErrorCode = "BAD_REQUEST"
	ErrCodeInvalidInput       ErrorCode = "INVALID_INPUT"
	ErrCodeInternalError      ErrorCode = "INTERNAL_ERROR"
	ErrCodeDatabaseError      ErrorCode = "DATABASE_ERROR"
	ErrCodeThrottling         ErrorCode = "THROTTLING_ERROR"
	ErrCodeResourceNotFound   ErrorCode = "RESOURCE_NOT_FOUND"
	ErrCodeValidationError    ErrorCode = "VALIDATION_ERROR"
	ErrCodeConditionalCheckFailed ErrorCode = "CONDITIONAL_CHECK_FAILED"
)

// AppError represents an application error with HTTP status and error code
type AppError struct {
	StatusCode int
	Code       ErrorCode
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// NewAppError creates a new application error
func NewAppError(statusCode int, code ErrorCode, message string, err error) *AppError {
	return &AppError{
		StatusCode: statusCode,
		Code:       code,
		Message:    message,
		Err:        err,
	}
}

// HandleError is a comprehensive error handler for DynamoDB and application errors
func HandleError(c *gin.Context, err error) {
	var appErr *AppError

	// Check if it's already an AppError
	if errors.As(err, &appErr) {
		log.Printf("ERROR [%s]: %v", appErr.Code, appErr.Error())
		c.JSON(appErr.StatusCode, ErrorResponse{
			Error:   string(appErr.Code),
			Message: appErr.Message,
		})
		return
	}

	// Handle known application errors
	switch {
	case errors.Is(err, ErrCartNotFound):
		log.Printf("WARN: Cart not found")
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   string(ErrCodeNotFound),
			Message: "Shopping cart not found",
		})
		return

	case errors.Is(err, ErrInvalidInput):
		log.Printf("WARN: Invalid input: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   string(ErrCodeInvalidInput),
			Message: "Invalid input provided",
		})
		return
	}

	// Handle DynamoDB-specific errors
	var (
		rnfe *types.ResourceNotFoundException
		pte  *types.ProvisionedThroughputExceededException
		ccfe *types.ConditionalCheckFailedException
		ise  *types.InternalServerError
		rle  *types.RequestLimitExceeded
		tme  *types.TransactionConflictException
	)

	// Check error message for validation errors (SDK v2 doesn't export ValidationException directly)
	errMsg := err.Error()
	isValidationError := strings.Contains(errMsg, "ValidationException") || 
		strings.Contains(errMsg, "invalid") || 
		strings.Contains(errMsg, "validation")

	switch {
	case isValidationError:
		// Validation error from DynamoDB
		log.Printf("ERROR: DynamoDB validation error: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   string(ErrCodeValidationError),
			Message: "Invalid data format for database operation",
		})

	case errors.As(err, &rnfe):
		// DynamoDB table or resource not found
		log.Printf("ERROR: DynamoDB resource not found: %v", err)
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   string(ErrCodeResourceNotFound),
			Message: "Database resource not found. Please contact support.",
		})

	case errors.As(err, &pte):
		// Throughput exceeded (throttling)
		log.Printf("ERROR: DynamoDB throttling: %v", err)
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error:   string(ErrCodeThrottling),
			Message: "Service is temporarily unavailable due to high load. Please try again.",
		})

	case errors.As(err, &ccfe):
		// Conditional check failed (item doesn't exist or condition not met)
		log.Printf("WARN: DynamoDB conditional check failed: %v", err)
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   string(ErrCodeConditionalCheckFailed),
			Message: "Shopping cart not found or condition not met",
		})

	case errors.As(err, &ise):
		// Internal server error from DynamoDB
		log.Printf("ERROR: DynamoDB internal server error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   string(ErrCodeDatabaseError),
			Message: "Database encountered an internal error. Please try again.",
		})

	case errors.As(err, &rle):
		// Request rate exceeded
		log.Printf("ERROR: DynamoDB request limit exceeded: %v", err)
		c.JSON(http.StatusTooManyRequests, ErrorResponse{
			Error:   string(ErrCodeThrottling),
			Message: "Too many requests. Please slow down and try again.",
		})

	case errors.As(err, &tme):
		// Transaction conflict
		log.Printf("ERROR: DynamoDB transaction conflict: %v", err)
		c.JSON(http.StatusConflict, ErrorResponse{
			Error:   string(ErrCodeDatabaseError),
			Message: "Database transaction conflict. Please try again.",
		})

	default:
		// Unknown error - log details but don't expose to client
		log.Printf("ERROR: Unexpected error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   string(ErrCodeInternalError),
			Message: "An unexpected error occurred. Please try again.",
		})
	}
}

// ValidationError creates a validation error
func ValidationError(message string) *AppError {
	return NewAppError(http.StatusBadRequest, ErrCodeInvalidInput, message, nil)
}

// NotFoundError creates a not found error
func NotFoundError(message string) *AppError {
	return NewAppError(http.StatusNotFound, ErrCodeNotFound, message, nil)
}

// InternalError creates an internal server error
func InternalError(message string, err error) *AppError {
	return NewAppError(http.StatusInternalServerError, ErrCodeInternalError, message, err)
}
