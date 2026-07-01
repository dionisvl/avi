package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"

	apierr "github.com/dionisvl/avi/api-go/internal/errors"
)

// ProblemDetails is RFC 9457 Problem Details response
type ProblemDetails struct {
	Status int    `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Code   string `json:"code"`
}

// PaginationResponse is common for all entities in project
type PaginationResponse struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

var validate = func() *validator.Validate {
	v := validator.New()
	// use json/query tag names in error messages instead of Go field names
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		if name := fld.Tag.Get("json"); name != "" && name != "-" {
			return strings.SplitN(name, ",", 2)[0]
		}
		if name := fld.Tag.Get("query"); name != "" && name != "-" {
			return name
		}
		return fld.Name
	})
	return v
}()

// validationDetail extracts a human-readable detail string from validator.ValidationErrors.
// Example: "condition: failed rule 'oneof'; price: failed rule 'min'"
func validationDetail(err error) string {
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return err.Error()
	}
	msgs := make([]string, 0, len(ve))
	for _, fe := range ve {
		msgs = append(msgs, fmt.Sprintf("%s: failed rule '%s'", fe.Field(), fe.Tag()))
	}
	return strings.Join(msgs, "; ")
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
	}
}

// Error writes RFC 9457 Problem Details response
func Error(w http.ResponseWriter, logger *slog.Logger, err error) {
	// Guard against nil error
	if err == nil {
		err = apierr.New(apierr.ErrInternal, "Internal Server Error")
	}

	// Find an AppError anywhere in the error chain (handles %w-wrapped values);
	// if there is none, wrap as internal server error.
	var appErr *apierr.AppError
	if !errors.As(err, &appErr) {
		logger.Error("unexpected non-AppError in api.Error", slog.String("error", err.Error()))
		appErr = apierr.Wrap(err, "Internal Server Error")
	}

	// Log internal error for debugging
	if appErr.Err != nil {
		logger.Error("request error", slog.String("error", appErr.Err.Error()))
	}

	resp := ProblemDetails{
		Status: appErr.Status,
		Title:  appErr.Title,
		Detail: appErr.Detail,
		Code:   string(appErr.Code),
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(appErr.Status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// DecodeQueryAndValidate populates T from URL query params using the `query` struct tag,
// then runs validator/v10 on the result. Returns ErrValidation on any failure.
func DecodeQueryAndValidate[T any](r *http.Request) (T, error) {
	var v T

	q := r.URL.Query()
	rv := reflect.ValueOf(&v).Elem()
	rt := rv.Type()

	for i := range rt.NumField() {
		field := rt.Field(i)
		tag := field.Tag.Get("query")
		if tag == "" || tag == "-" {
			continue
		}
		raw := q.Get(tag)
		if raw == "" {
			continue
		}
		fv := rv.Field(i)
		//nolint:exhaustive
		switch fv.Kind() {
		case reflect.Slice:
			parts := strings.Split(raw, ",")
			if fv.Type().Elem().Kind() == reflect.String {
				sl := reflect.MakeSlice(fv.Type(), len(parts), len(parts))
				for i, p := range parts {
					sl.Index(i).SetString(strings.TrimSpace(p))
				}
				fv.Set(sl)
			} else if fv.Type().Elem() == reflect.TypeFor[uuid.UUID]() {
				sl := reflect.MakeSlice(fv.Type(), 0, len(parts))
				for _, p := range parts {
					id, err := uuid.Parse(strings.TrimSpace(p))
					if err != nil {
						return v, apierr.New(apierr.ErrValidation, fmt.Sprintf("invalid value for %s: each item must be UUID", tag))
					}
					sl = reflect.Append(sl, reflect.ValueOf(id))
				}
				fv.Set(sl)
			}
		case reflect.String:
			fv.SetString(raw)
		case reflect.Struct, reflect.Array:
			if fv.Type() == reflect.TypeFor[uuid.UUID]() {
				id, err := uuid.Parse(raw)
				if err != nil {
					return v, apierr.New(apierr.ErrValidation, fmt.Sprintf("invalid value for %s: must be UUID", tag))
				}
				fv.Set(reflect.ValueOf(id))
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return v, apierr.New(apierr.ErrValidation, fmt.Sprintf("invalid value for %s: must be integer", tag))
			}
			fv.SetInt(n)
		case reflect.Ptr:
			//nolint:exhaustive
			switch fv.Type().Elem().Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				n, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					return v, apierr.New(apierr.ErrValidation, fmt.Sprintf("invalid value for %s: must be integer", tag))
				}
				ptr := reflect.New(fv.Type().Elem())
				ptr.Elem().SetInt(n)
				fv.Set(ptr)
			case reflect.String:
				s := raw
				fv.Set(reflect.ValueOf(&s))
			case reflect.Struct, reflect.Array:
				if fv.Type().Elem() == reflect.TypeFor[uuid.UUID]() {
					id, err := uuid.Parse(raw)
					if err != nil {
						return v, apierr.New(apierr.ErrValidation, fmt.Sprintf("invalid value for %s: must be UUID", tag))
					}
					ptr := reflect.New(fv.Type().Elem())
					ptr.Elem().Set(reflect.ValueOf(id))
					fv.Set(ptr)
				}
			default:
				// unsupported pointer elem kind, skip
			}
		default:
			// unsupported kind, skip
		}
	}

	if err := validate.Struct(v); err != nil {
		return v, apierr.New(apierr.ErrValidation, validationDetail(err))
	}

	return v, nil
}

const maxJSONBodyBytes = 1 << 20 // 1 MiB

func DecodeAndValidate[T any](r *http.Request) (T, error) {
	var v T

	r.Body = http.MaxBytesReader(nil, r.Body, maxJSONBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return v, apierr.New(apierr.ErrRequestTooLarge, "Request body too large")
		}
		return v, apierr.New(apierr.ErrBadRequest, "Failed to read request body")
	}

	if err := json.Unmarshal(body, &v); err != nil {
		return v, apierr.New(apierr.ErrBadRequest, "Invalid JSON in request body")
	}

	if err := validate.Struct(v); err != nil {
		return v, apierr.New(apierr.ErrValidation, validationDetail(err))
	}

	return v, nil
}

func ValidateStruct(v any) error {
	if err := validate.Struct(v); err != nil {
		return apierr.New(apierr.ErrValidation, validationDetail(err))
	}
	return nil
}

func AsAppError(err error) *apierr.AppError {
	if err == nil {
		return nil
	}

	var appErr *apierr.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	return apierr.Wrap(err, "Internal server error")
}

func ValidationError(w http.ResponseWriter, logger *slog.Logger, msg string) {
	Error(w, logger, apierr.New(apierr.ErrValidation, msg))
}

// WriteError writes an RFC 9457 Problem Details response without logging.
// Use in middleware or places where no logger is available.
func WriteError(w http.ResponseWriter, appErr *apierr.AppError) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(ProblemDetails{
		Status: appErr.Status,
		Title:  appErr.Title,
		Detail: appErr.Detail,
		Code:   string(appErr.Code),
	})
}
