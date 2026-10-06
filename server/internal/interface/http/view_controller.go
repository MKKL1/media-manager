package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"server/internal/plugin"
	"server/internal/view"
)

type ViewController struct {
	Views *view.Views
}

func (c *ViewController) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/entries/{id}/view", c.View)
}

func (c *ViewController) View(w http.ResponseWriter, r *http.Request) {
	id, err := entryID(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, err)
		return
	}
	q := r.URL.Query()
	shown, err := c.Views.Show(r.Context(), id, view.Arrangement(q.Get("arrangement")), plugin.Tag(q.Get("type")))
	if err != nil {
		RespondError(w, r, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": shown})
}
