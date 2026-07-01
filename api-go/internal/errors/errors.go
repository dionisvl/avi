package errors

import (
	"errors"
	"net/http"
)

// Sentinel errors for business logic
var (
	// Auth errors
	ErrUserNotFound            = errors.New("user not found")
	ErrUserAlreadyExists       = errors.New("user already exists")
	ErrUserExistsUnverified    = errors.New("user exists but email not verified")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrInvalidToken            = errors.New("invalid token")
	ErrTokenExpired            = errors.New("token expired")
	ErrEmailNotVerified        = errors.New("email not verified")
	ErrInvalidVerificationCode = errors.New("invalid verification code")
	ErrEmailAlreadyVerified    = errors.New("email already verified")
	ErrInvalidResetCode        = errors.New("invalid or expired reset code")

	// Validation errors
	ErrValidation      = errors.New("validation failed")
	ErrBadRequest      = errors.New("bad request")
	ErrRequestTooLarge = errors.New("request too large")

	// Resource errors
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")

	// Generic errors
	ErrForbidden   = errors.New("forbidden")
	ErrRateLimited = errors.New("rate limit exceeded")
	ErrInternal    = errors.New("internal server error")
)

// AppError is RFC 9457 Problem Details response
type AppError struct {
	Status int       // HTTP status code
	Title  string    // short label, e.g. "Validation Failed"
	Detail string    // safe human-readable explanation
	Code   ErrorCode // stable machine-readable error code
	Err    error     // internal cause, never expose to client
}

type ErrorCode string

const (
	CodeUserNotFound            ErrorCode = "USER_NOT_FOUND"
	CodeUserAlreadyExists       ErrorCode = "USER_ALREADY_EXISTS"
	CodeUserExistsUnverified    ErrorCode = "USER_EXISTS_UNVERIFIED"
	CodeInvalidCredentials      ErrorCode = "INVALID_CREDENTIALS"
	CodeInvalidToken            ErrorCode = "INVALID_TOKEN"
	CodeTokenExpired            ErrorCode = "TOKEN_EXPIRED"
	CodeEmailNotVerified        ErrorCode = "EMAIL_NOT_VERIFIED"
	CodeInvalidVerificationCode ErrorCode = "INVALID_VERIFICATION_CODE"
	CodeEmailAlreadyVerified    ErrorCode = "EMAIL_ALREADY_VERIFIED"
	CodeInvalidResetCode        ErrorCode = "INVALID_RESET_CODE"
	CodeValidationFailed        ErrorCode = "VALIDATION_FAILED"
	CodeBadRequest              ErrorCode = "BAD_REQUEST"
	CodeRequestTooLarge         ErrorCode = "REQUEST_TOO_LARGE"
	CodeNotFound                ErrorCode = "NOT_FOUND"
	CodeAlreadyExists           ErrorCode = "ALREADY_EXISTS"
	CodeForbidden               ErrorCode = "FORBIDDEN"
	CodeRateLimited             ErrorCode = "RATE_LIMITED"
	CodeInternalServerError     ErrorCode = "INTERNAL_SERVER_ERROR"

	CodeRequestStatusOutdated  ErrorCode = "REQUEST_STATUS_OUTDATED"
	CodeWrongCurrentPassword   ErrorCode = "WRONG_CURRENT_PASSWORD"
	CodeIncompleteOwnerProfile ErrorCode = "INCOMPLETE_OWNER_PROFILE"
)

type problemClass struct {
	status int
	title  string
	code   ErrorCode
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Title
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates an AppError from a sentinel error with safe client message
func New(sentinelErr error, clientDetail string) *AppError {
	class := classify(sentinelErr)

	return &AppError{
		Err:    sentinelErr,
		Detail: clientDetail,
		Status: class.status,
		Title:  class.title,
		Code:   class.code,
	}
}

// NewWithCode creates an AppError with a more specific machine-readable code.
func NewWithCode(sentinelErr error, code ErrorCode, clientDetail string) *AppError {
	appErr := New(sentinelErr, clientDetail)
	appErr.Code = code
	return appErr
}

// Wrap wraps an error with additional context
func Wrap(err error, clientDetail string) *AppError {
	return &AppError{
		Err:    err,
		Detail: clientDetail,
		Status: http.StatusInternalServerError,
		Title:  "Internal Server Error",
		Code:   CodeInternalServerError,
	}
}

var problemClasses = []struct {
	target error
	class  problemClass
}{
	{target: ErrUserNotFound, class: problemClass{status: http.StatusNotFound, title: "Not Found", code: CodeUserNotFound}},
	{target: ErrUserAlreadyExists, class: problemClass{status: http.StatusConflict, title: "Conflict", code: CodeUserAlreadyExists}},
	{target: ErrUserExistsUnverified, class: problemClass{status: http.StatusConflict, title: "Conflict", code: CodeUserExistsUnverified}},
	{target: ErrInvalidCredentials, class: problemClass{status: http.StatusUnauthorized, title: "Unauthorized", code: CodeInvalidCredentials}},
	{target: ErrEmailNotVerified, class: problemClass{status: http.StatusUnauthorized, title: "Unauthorized", code: CodeEmailNotVerified}},
	{target: ErrInvalidToken, class: problemClass{status: http.StatusUnauthorized, title: "Unauthorized", code: CodeInvalidToken}},
	{target: ErrTokenExpired, class: problemClass{status: http.StatusUnauthorized, title: "Unauthorized", code: CodeTokenExpired}},
	{target: ErrInvalidVerificationCode, class: problemClass{status: http.StatusBadRequest, title: "Invalid Verification Code", code: CodeInvalidVerificationCode}},
	{target: ErrEmailAlreadyVerified, class: problemClass{status: http.StatusBadRequest, title: "Email Already Verified", code: CodeEmailAlreadyVerified}},
	{target: ErrInvalidResetCode, class: problemClass{status: http.StatusBadRequest, title: "Invalid Reset Code", code: CodeInvalidResetCode}},
	{target: ErrNotFound, class: problemClass{status: http.StatusNotFound, title: "Not Found", code: CodeNotFound}},
	{target: ErrAlreadyExists, class: problemClass{status: http.StatusConflict, title: "Conflict", code: CodeAlreadyExists}},
	{target: ErrValidation, class: problemClass{status: http.StatusBadRequest, title: "Validation Failed", code: CodeValidationFailed}},
	{target: ErrBadRequest, class: problemClass{status: http.StatusBadRequest, title: "Bad Request", code: CodeBadRequest}},
	{target: ErrRequestTooLarge, class: problemClass{status: http.StatusRequestEntityTooLarge, title: "Request Too Large", code: CodeRequestTooLarge}},
	{target: ErrForbidden, class: problemClass{status: http.StatusForbidden, title: "Forbidden", code: CodeForbidden}},
	{target: ErrRateLimited, class: problemClass{status: http.StatusTooManyRequests, title: "Too Many Requests", code: CodeRateLimited}},
	{target: ErrInternal, class: problemClass{status: http.StatusInternalServerError, title: "Internal Server Error", code: CodeInternalServerError}},
}

func classify(err error) problemClass {
	for _, candidate := range problemClasses {
		if errors.Is(err, candidate.target) {
			return candidate.class
		}
	}

	return problemClass{
		status: http.StatusInternalServerError,
		title:  "Internal Server Error",
		code:   CodeInternalServerError,
	}
}
