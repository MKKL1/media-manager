package http

import (
	"errors"
	"net/http"
	"server/internal/domain"

	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		panic(err)
	}
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	logger := zerolog.Ctx(r.Context())

	switch {
	case errors.Is(err, domain.ErrNotFound):
		JSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
	case errors.Is(err, domain.ErrAlreadyExists):
		JSON(w, http.StatusConflict, map[string]string{"error": "already exists"})
	case errors.Is(err, domain.ErrInvalidInput):
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, domain.ErrNoProvider):
		JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	default:
		logger.Error().Err(err).Msg("unhandled http request error")
		JSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}
}
