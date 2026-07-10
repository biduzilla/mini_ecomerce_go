package apiError

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"ms_order/internal/core/contexts"
	"ms_order/internal/core/jsonlog"
	"ms_order/pkg/httpjson"
)

type ErrorHandler struct {
	logger jsonlog.Logger
}

type ValidationError struct {
	FieldErrors map[string]string
}

type ApiError struct {
	Message string
	Code    int
}

func (e *ApiError) Error() string {
	return e.Message
}

func (e *ValidationError) Error() string {
	return "validation failed"
}

type DetailedApiError struct {
	Message string
	Code    int
	Details map[string]string
}

func (e *DetailedApiError) Error() string {
	return e.Message
}

func NewDetailedApiError(message string, code int, details map[string]string) *DetailedApiError {
	return &DetailedApiError{
		Message: message,
		Code:    code,
		Details: details,
	}
}

func NewErrorHandler(logger jsonlog.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

var (
	ErrRecordNotFound           = errors.New("record not found")
	ErrEditConflict             = errors.New("edit conflict")
	ErrInvalidData              = errors.New("invalid data")
	ErrInvalidCredentials       = errors.New("invalid authentication credentials")
	ErrTokenExpired             = errors.New("token has expired")
	ErrInvalidTokenType         = errors.New("invalid token type for this operation")
	ErrInvalidTokenClaims       = errors.New("token claims are invalid or malformed")
	ErrInvalidTokenSignature    = errors.New("token signature invalid")
	ErrCountPermissions         = errors.New("one or more permissions do not exist")
	ErrRollbackFailed           = errors.New("transaction rollback failed")
	ErrTransactionNotFound      = errors.New("transaction not found")
	ErrInactiveAccount          = errors.New("your user account must be activated to access this resource")
	ErrStartDateAfterEndDate    = errors.New("start date must be before end date")
	ErrInvalidRole              = errors.New("invalid role")
	ErrScanModel                = errors.New("dest must be a pointer")
	ErrUnsupportedTypeScanModel = errors.New("unsupported slice type for db scan")
)

func (e *ErrorHandler) HandlerError(w http.ResponseWriter, r *http.Request, err error) {
	var valErr *ValidationError
	var apiError *ApiError
	var detailedErr *DetailedApiError

	switch {
	case errors.As(err, &valErr):
		e.FailedValidationResponse(w, r, valErr.FieldErrors)
	case errors.As(err, &apiError):
		e.errorHandler(w, r, apiError.Code, apiError.Message)
	case errors.As(err, &detailedErr): // NOVO
		e.DetailedResponse(w, r, detailedErr)

	case errors.Is(err, context.DeadlineExceeded):
		e.RequestTimeoutResponse(w, r)

	case errors.Is(err, ErrRecordNotFound):
		e.NotFoundResponse(w, r)

	case errors.Is(err, ErrEditConflict):
		e.EditConflictResponse(w, r)

	case errors.Is(err, ErrInactiveAccount):
		e.InactiveAccountResponse(w, r)

	case errors.Is(err, ErrInvalidRole):
		e.InvalidRoleResponse(w, r)

	case errors.Is(err, ErrInvalidCredentials):
		e.InvalidCredentialsResponse(w, r)

	case errors.Is(err, ErrTokenExpired) ||
		errors.Is(err, ErrInvalidTokenType) ||
		errors.Is(err, ErrInvalidTokenClaims):
		e.TokenErrorResponse(w, r, err)

	default:
		e.ServerErrorResponse(w, r, err)
	}
}

func ValidationAlreadyExists(field string) error {
	return &ValidationError{
		FieldErrors: map[string]string{
			field: fmt.Sprintf("a record with this %s already exists", field),
		},
	}
}

func NewValidationError(fieldErrors map[string]string) *ValidationError {
	return &ValidationError{
		FieldErrors: fieldErrors,
	}
}

func NewApiError(message string, code int) *ApiError {
	return &ApiError{
		Message: message,
		Code:    code,
	}
}

func (e *ErrorHandler) DetailedResponse(w http.ResponseWriter, r *http.Request, err *DetailedApiError) {
	payload := map[string]any{
		"status":  http.StatusText(err.Code),
		"message": err.Message,
		"details": err.Details,
	}

	writeErr := httpjson.WriteJSON(w, err.Code, payload, nil)
	if writeErr != nil {
		e.logError(r, writeErr)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (e *ErrorHandler) NotPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	e.errorHandler(w, r, http.StatusForbidden, message)
}

func (e *ErrorHandler) RequestTimeoutResponse(w http.ResponseWriter, r *http.Request) {
	message := "request timeout, please try again"
	e.logger.PrintInfo("request timeout", map[string]string{
		"method":     r.Method,
		"path":       r.URL.Path,
		"ip":         r.RemoteAddr,
		"request_id": contexts.GetRequestID(r.Context()),
	})
	e.errorHandler(w, r, http.StatusGatewayTimeout, message)
}

func (e *ErrorHandler) MalFormedTokenResponse(w http.ResponseWriter, r *http.Request) {
	message := "malformed token"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *ErrorHandler) TokenErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.logError(r, err)
	message := "invalid or malformed authentication token"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *ErrorHandler) ExpiredTokenResponse(w http.ResponseWriter, r *http.Request) {
	message := "expired token"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *ErrorHandler) AuthenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}
func (e *ErrorHandler) InactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	e.errorHandler(w, r, http.StatusForbidden, message)
}

func (e *ErrorHandler) InvalidRoleResponse(w http.ResponseWriter, r *http.Request) {
	message := "Your user account does not have access to this feature."
	e.errorHandler(w, r, http.StatusForbidden, message)
}

func (e *ErrorHandler) InvalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *ErrorHandler) InvalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	e.errorHandler(w, r, http.StatusUnauthorized, message)
}

func (e *ErrorHandler) RateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceed"
	e.errorHandler(w, r, http.StatusTooManyRequests, message)
}

func (e *ErrorHandler) ServerErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	e.errorHandler(w, r, http.StatusInternalServerError, message)
}

func (e *ErrorHandler) NotFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	e.errorHandler(w, r, http.StatusNotFound, message)
}

func (e *ErrorHandler) MethodNotAllowedResponse(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	e.errorHandler(w, r, http.StatusMethodNotAllowed, message)
}

func (e *ErrorHandler) BadRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	e.errorHandler(w, r, http.StatusBadRequest, err.Error())
}

func (e *ErrorHandler) FailedValidationResponse(w http.ResponseWriter, r *http.Request, fieldErrors map[string]string) {
	payload := map[string]any{
		"path":    r.URL.Path,
		"status":  http.StatusText(http.StatusUnprocessableEntity),
		"message": "validation failed",
		"errors":  fieldErrors,
	}
	err := httpjson.WriteJSON(w, http.StatusUnprocessableEntity, payload, nil)
	if err != nil {
		e.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (e *ErrorHandler) EditConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	e.errorHandler(w, r, http.StatusConflict, message)
}

func (e *ErrorHandler) errorHandler(w http.ResponseWriter, r *http.Request, status int, message string) {
	err := httpjson.WriteJSON(w, status, map[string]any{
		"path":    r.URL.Path,
		"status":  http.StatusText(status),
		"message": message,
	}, nil)
	if err != nil {
		e.logError(r, err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (e *ErrorHandler) logError(r *http.Request, err error) {
	e.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
		"request_id":     contexts.GetRequestID(r.Context()),
	})
}
