package http

import (
	"net/http"
	"server/internal/core"
	"server/internal/metadata/services"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"github.com/rs/zerolog"
)

type MediaController struct {
	pullService *services.PullService
	logger      zerolog.Logger
}

func NewMediaController(pullService *services.PullService, logger zerolog.Logger) *MediaController {
	return &MediaController{pullService: pullService, logger: logger}
}

func (c *MediaController) Route(r *chi.Mux) http.Handler {
	return r.Route("/api/1/media", func(r chi.Router) {
		r.Post("/pull", c.PullMedia)
	})
}

type pullMediaRequest struct {
	Provider  string         `json:"provider"`
	ID        string         `json:"id"`
	MediaType core.MediaType `json:"media_type"`
}

func (req pullMediaRequest) validate() error {
	if req.Provider == "" || req.ID == "" {
		return core.ErrInvalidInput
	}
	if req.MediaType != "movie" && req.MediaType != "tv" {
		return core.ErrInvalidInput
	}
	return nil
}

func (c *MediaController) PullMedia(w http.ResponseWriter, r *http.Request) {
	var req pullMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	if err := req.validate(); err != nil {
		Error(w, r, err)
		return
	}

	extID := core.NewExternalId(req.Provider, req.ID)
	task, err := c.pullService.RequestPull(r.Context(), extID, req.MediaType)
	if err != nil {
		Error(w, r, err)
		return
	}

	JSON(w, http.StatusAccepted, map[string]any{"status": "queued", "task": task})
}
