package http

import (
	"context"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
)

type PluginController struct {
	Plugins interface {
		Names() []string
		Reload(ctx context.Context, name string) error
	}
}

func (c *PluginController) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/plugins", c.List)
	r.Post("/api/v1/plugins/{name}/reload", c.Reload)
}

func (c *PluginController) List(w http.ResponseWriter, r *http.Request) {
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": c.Plugins.Names()})
}

func (c *PluginController) Reload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if !slices.Contains(c.Plugins.Names(), name) {
		RespondError(w, r, errNotFound)
		return
	}
	if err := c.Plugins.Reload(r.Context(), name); err != nil {
		RespondError(w, r, err)
		return
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"reloaded": name}})
}
