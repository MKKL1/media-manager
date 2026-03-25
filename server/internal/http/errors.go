package http

import (
	"errors"
	"net/http"
	"server/internal/domain"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
)

//var domainErrors = []struct {
//	target error
//	status int
//	code   string
//	msg    string
//}{
//	{domain.ErrNotFound, http.StatusNotFound, "not_found", "resource not found"},
//	{domain.ErrAlreadyExists, http.StatusConflict, "conflict", "resource already exists"},
//	{domain.ErrInvalidInput, http.StatusBadRequest, "bad_request", "resource already exists"},
//	{domain.ErrNoProvider, http.StatusBadRequest, "no_provider", "resource already exists"},
//}

// APIError is the stable error shape every endpoint returns.
type APIError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	if ve, ok := errors.AsType[validator.ValidationErrors](err); ok {
		fields := make([]FieldError, len(ve))
		for i, fe := range ve {
			fields[i] = FieldError{
				Field:   fe.Field(),
				Rule:    fe.Tag(),
				Message: fieldMessage(fe),
			}
		}
		writeJSON(w, http.StatusUnprocessableEntity, errorEnvelope{
			Error: APIError{
				Code:    "validation_failed",
				Message: "one or more fields failed validation",
				Fields:  fields,
			},
		})
		return
	}

	code, status, msg := "internal", http.StatusInternalServerError, "internal server error"

	switch {
	case errors.Is(err, domain.ErrNotFound):
		code, status, msg = "not_found", http.StatusNotFound, "resource not found"
	case errors.Is(err, domain.ErrAlreadyExists):
		code, status, msg = "conflict", http.StatusConflict, "resource already exists"
	case errors.Is(err, domain.ErrInvalidInput):
		code, status, msg = "bad_request", http.StatusBadRequest, err.Error()
	case errors.Is(err, domain.ErrNoProvider):
		code, status, msg = "no_provider", http.StatusBadRequest, err.Error()
	default:
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("unhandled error")
	}

	writeJSON(w, status, errorEnvelope{
		Error: APIError{Code: code, Message: msg},
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "min":
		return "value is too small"
	case "max":
		return "value is too large"
	default:
		return "invalid value"
	}
}
