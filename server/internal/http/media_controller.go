package http

import (
	"net/http"
	"server/internal/domain"
	"server/internal/metadata"

	"github.com/go-chi/chi/v5"
)

type MediaController struct {
	pullService *metadata.PullService
	mdService   *metadata.Service
}

func NewMediaController(pullService *metadata.PullService, mdService *metadata.Service) *MediaController {
	return &MediaController{pullService: pullService, mdService: mdService}
}

func (c *MediaController) Route(r *chi.Mux) {
	r.Route("/api/1/media", func(r chi.Router) {
		r.Post("/pull", c.PullMedia)
		r.Get("/list", c.List)
	})
}

func (c *MediaController) List(w http.ResponseWriter, r *http.Request) {
	var req queryMediaRequest
	if err := decodeQuery(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}

	list, err := c.mdService.List(r.Context(), req.ToDomain())
	if err != nil {
		RespondError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": list})
}

func (c *MediaController) PullMedia(w http.ResponseWriter, r *http.Request) {
	var req pullMediaRequest
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}

	extID := domain.NewExternalId(req.Provider, req.ID)
	instanceID, err := c.pullService.RequestPull(r.Context(), extID, req.MediaType)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":      "queued",
		"workflow_id": instanceID,
	})
}
