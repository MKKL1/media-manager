package http

import (
	"net/http"
	"server/internal/domain"
	"server/internal/metadata"

	"github.com/go-chi/chi/v5"
)

type MediaController struct {
	pullService   *metadata.PullService
	mdService     *metadata.Service
	searchService *metadata.SearchService
}

func NewMediaController(
	pullService *metadata.PullService,
	mdService *metadata.Service,
	searchService *metadata.SearchService,
) *MediaController {
	return &MediaController{
		pullService:   pullService,
		mdService:     mdService,
		searchService: searchService,
	}
}

func (c *MediaController) Route(r *chi.Mux) {
	r.Route("/api/1/media", func(r chi.Router) {
		r.Get("/:id", c.GetMedia)
		r.Post("/pull", c.PullMedia)
		r.Get("/list", c.List)
		r.Get("/search", c.Search)
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

	var items = make([]MediaSummaryResponse, 0, len(list.Items))
	for _, item := range list.Items {
		items = append(items, toMediaSummaryResponse(item))
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": MediaPageResponse{
		Items:  items,
		Total:  list.Total,
		Offset: list.Offset,
		Limit:  list.Limit,
	}})
}

func (c *MediaController) PullMedia(w http.ResponseWriter, r *http.Request) {
	var req pullMediaRequest
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}

	//TODO domain.ProviderKind(req.Provider) should be validated
	extID := domain.NewMediaIdentity(domain.SourceKindFromString(req.Provider), req.ID)
	instanceID, err := c.pullService.RequestPull(r.Context(), extID, req.MediaType)
	if err != nil {
		RespondError(w, r, err)
		return
	}

	writeJSON(w, http.StatusAccepted, PullMediaResponse{
		Status:     "queued",
		WorkflowID: instanceID,
	})
}

func (c *MediaController) Search(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := decodeQuery(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}

	results, err := c.searchService.Search(r.Context(), domain.SearchQuery{
		Query:    req.Query,
		Year:     req.Year,
		Language: req.Language,
	})
	if err != nil {
		RespondError(w, r, err)
		return
	}

	var data = make([]SearchResultResponse, 0, len(results))
	for _, item := range results {
		data = append(data, toSearchResultResponse(item))
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func (c *MediaController) GetMedia(w http.ResponseWriter, r *http.Request) {

	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}
