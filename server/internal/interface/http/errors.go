package http

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
	"server/internal/user"
	"server/internal/view"
)

type APIError struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
	Details any          `json:"details,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type errorEnvelope struct {
	Error APIError `json:"error"`
}

var errBadRequest = errors.New("malformed request")

var errNotFound = errors.New("not found")

func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		fields := make([]FieldError, len(ve))
		for i, fe := range ve {
			fields[i] = FieldError{
				Field:   fe.Field(),
				Rule:    fe.Tag(),
				Message: fieldMessage(fe),
			}
		}
		_ = writeJSON(w, http.StatusUnprocessableEntity, errorEnvelope{
			Error: APIError{
				Code:    "validation_failed",
				Message: "one or more fields failed validation",
				Fields:  fields,
			},
		})
		return
	}

	var nf view.NotFound
	if errors.As(err, &nf) {
		_ = writeJSON(w, http.StatusNotFound, errorEnvelope{Error: APIError{
			Code: "not_found", Message: err.Error(),
			Details: map[string]string{"kind": string(nf.Kind), "name": nf.Name},
		}})
		return
	}

	code, status, msg := "internal", http.StatusInternalServerError, "internal server error"

	switch {
	case errors.Is(err, errNotFound), errors.Is(err, plugin.ErrNotFound), errors.Is(err, plugin.ErrNoSuchArrangement), errors.Is(err, user.ErrNotFound):
		code, status, msg = "not_found", http.StatusNotFound, err.Error()
	case errors.Is(err, user.ErrUnauthenticated):
		code, status, msg = "unauthenticated", http.StatusUnauthorized, err.Error()
	case errors.Is(err, user.ErrForbidden):
		code, status, msg = "forbidden", http.StatusForbidden, err.Error()
	case errors.Is(err, user.ErrInvalid):
		code, status, msg = "invalid", http.StatusBadRequest, err.Error()
	case errors.Is(err, errors.ErrUnsupported):
		code, status, msg = "not_offered", http.StatusNotImplemented, err.Error()
	case errors.Is(err, library.ErrRefused):
		code, status, msg = "refused", http.StatusConflict, err.Error()
	case errors.Is(err, plugin.ErrNotLoaded):
		code, status, msg = "not_loaded", http.StatusUnprocessableEntity, err.Error()
	case errors.Is(err, event.ErrTooOld):
		code, status, msg = "gone", http.StatusGone, err.Error()
	case errors.Is(err, errBadRequest):
		code, status, msg = "bad_request", http.StatusBadRequest, err.Error()
	default:
		zerolog.Ctx(r.Context()).Error().Err(err).Msg("unhandled error")
	}

	_ = writeJSON(w, status, errorEnvelope{Error: APIError{Code: code, Message: msg}})
}

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	data, err := sonic.Marshal(v)
	if err != nil {
		return fmt.Errorf("sonic.Marshal: %w", err)
	}
	_, err = w.Write(data)
	return err
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
