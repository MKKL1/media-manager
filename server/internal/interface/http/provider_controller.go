package http

import (
	"context"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"

	"server/internal/library"
	"server/internal/plugin"
)

type ProviderController struct {
	Library  *library.Library
	Plugins  func() []plugin.Plugin
	Mappings []plugin.Mappings
}

type refLister interface {
	Refs(ctx context.Context, lib *library.Library) ([]string, error)
}

func (c *ProviderController) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/providers", c.List)
	r.Get("/api/v1/providers/{provider}/search", c.Search)
	r.Post("/api/v1/providers/{provider}/fetch", c.Fetch)
}

func (c *ProviderController) plugin(name string) (plugin.Plugin, bool) {
	for _, p := range c.Plugins() {
		if string(p.Provider()) == name {
			return p, true
		}
	}
	return nil, false
}

type providerResponse struct {
	Provider     string       `json:"provider"`
	Arrangements []string     `json:"arrangements"`
	CanSearch    bool         `json:"can_search"`
	Tags         []plugin.Tag `json:"tags"`
}

func (c *ProviderController) List(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query().Get("tag")
	resp := []providerResponse{}
	for _, p := range c.Plugins() {
		tags := plugin.TagsOf(p)
		if tag != "" && !slices.Contains(tags, plugin.Tag(tag)) {
			continue
		}
		_, canSearch := p.(plugin.Searcher)
		resp = append(resp, providerResponse{string(p.Provider()), plugin.ArrangementsOf(p), canSearch, tags})
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

type searchResultResponse struct {
	Ref    string            `json:"ref"`
	Values map[string]string `json:"values"`
}

func (c *ProviderController) Search(w http.ResponseWriter, r *http.Request) {
	p, ok := c.plugin(chi.URLParam(r, "provider"))
	if !ok {
		RespondError(w, r, errNotFound)
		return
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		RespondError(w, r, errBadRequest)
		return
	}
	found, err := plugin.Search(r.Context(), p, query)
	if err != nil {
		RespondError(w, r, err)
		return
	}
	resp := make([]searchResultResponse, len(found))
	for i, f := range found {
		resp[i] = searchResultResponse{Ref: f.Ref, Values: map[string]string{}}
		for field, v := range f.Values {
			resp[i].Values[string(field)] = v
		}
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

type fetchRequest struct {
	Ref         string `json:"ref"`
	Arrangement string `json:"arrangement"`
}

func (c *ProviderController) Fetch(w http.ResponseWriter, r *http.Request) {
	var req fetchRequest
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	ctx, name := r.Context(), library.Provider(chi.URLParam(r, "provider"))
	if p, ok := c.plugin(string(name)); ok {
		if req.Ref == "" {
			RespondError(w, r, errBadRequest)
			return
		}
		top, err := plugin.Fetch(ctx, c.Library, p, req.Arrangement, req.Ref)
		if err != nil {
			RespondError(w, r, err)
			return
		}
		_ = writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"fetched": []string{top.String()}}})
		return
	}
	for _, p := range c.Mappings {
		if p.Provider() != name {
			continue
		}
		refs := []string{req.Ref}
		if lister, ok := p.(refLister); ok && req.Ref == "" {
			var err error
			if refs, err = lister.Refs(ctx, c.Library); err != nil {
				RespondError(w, r, err)
				return
			}
		}
		for _, ref := range refs {
			if err := plugin.FetchMapping(ctx, c.Library, p, ref); err != nil {
				RespondError(w, r, err)
				return
			}
		}
		_ = writeJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"fetched": refs}})
		return
	}
	RespondError(w, r, errNotFound)
}
