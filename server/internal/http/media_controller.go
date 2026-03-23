package http

import (
	"net/http"
	"server/internal/domain"
	"server/internal/metadata"

	"github.com/go-chi/chi/v5"
	"github.com/goccy/go-json"
	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("http/media")

type MediaController struct {
	pullService *metadata.PullService
}

func NewMediaController(pullService *metadata.PullService) *MediaController {
	return &MediaController{pullService: pullService}
}

func (c *MediaController) Route(r *chi.Mux) http.Handler {
	return r.Route("/api/1/media", func(r chi.Router) {
		r.Post("/pull", c.PullMedia)
	})
}

type pullMediaRequest struct {
	Provider  string           `json:"provider"`
	ID        string           `json:"id"`
	MediaType domain.MediaType `json:"media_type"`
}

func (req pullMediaRequest) validate() error {
	if req.Provider == "" || req.ID == "" {
		return domain.ErrInvalidInput
	}
	if req.MediaType != "movie" && req.MediaType != "tv" {
		return domain.ErrInvalidInput
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

	extID := domain.NewExternalId(req.Provider, req.ID)
	instanceID, err := c.pullService.RequestPull(r.Context(), extID, req.MediaType)
	if err != nil {
		Error(w, r, err)
		return
	}

	JSON(w, http.StatusAccepted, map[string]any{
		"status":      "queued",
		"workflow_id": instanceID,
	})
}
