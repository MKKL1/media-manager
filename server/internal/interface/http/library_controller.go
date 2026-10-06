package http

import (
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"server/internal/library"
)

var validate = validator.New()

func decodeJSON(r *http.Request, dst any) error {
	if err := sonic.ConfigDefault.NewDecoder(r.Body).Decode(dst); err != nil {
		return errBadRequest
	}
	return validate.Struct(dst)
}

var Fields = []library.Field{
	"title", "original_title", "summary", "status", "genres", "original_language",
	"air_date", "first_air_date", "release_date",
	"season_number", "episode_number", "absolute_number", "order",
	"id.tmdb", "id.tvdb", "id.imdb", "tags",
}

type LibraryController struct {
	Library *library.Library
}

func (c *LibraryController) RegisterRoutes(r chi.Router) {
	r.Get("/api/v1/library", c.Overview)
	r.Get("/api/v1/entries/{id}", c.GetEntry)
	r.Post("/api/v1/entries/{id}/edits", c.Edit)
}

func entryID(s string) (library.ID, error) {
	p, id, ok := strings.Cut(s, ":")
	if !ok || p == "" || id == "" {
		return library.ID{}, errBadRequest
	}
	return library.ID{Provider: library.Provider(p), ProviderID: id}, nil
}

func (c *LibraryController) Overview(w http.ResponseWriter, r *http.Request) {
	lib := c.Library
	resp := overviewResponse{Works: []summaryResponse{}}
	for e := range lib.Works() {
		resp.Works = append(resp.Works, summary(e))
	}
	for report := range lib.Reports() {
		rr := reportResponse{Kind: string(report.Kind)}
		for _, e := range report.Entries {
			rr.Entries = append(rr.Entries, e.ID().String())
		}
		resp.Reports = append(resp.Reports, rr)
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

func (c *LibraryController) GetEntry(w http.ResponseWriter, r *http.Request) {
	id, err := entryID(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, err)
		return
	}
	e, ok := c.Library.Entry(id)
	if !ok {
		RespondError(w, r, errNotFound)
		return
	}
	resp := entryResponse{ID: id.String(), Values: map[string][]valueResponse{}}
	for _, f := range Fields {
		for s, text := range e.Values(f) {
			v := valueResponse{Text: text}
			switch s := s.(type) {
			case library.Value:
				v.Edited = true
			case library.Fetched:
				v.Arrangement = s.Arrangement
			}
			resp.Values[string(f)] = append(resp.Values[string(f)], v)
		}
	}
	if g, ok := e.AsGroup(); ok {
		resp.Entries = []summaryResponse{}
		for child := range g.Entries() {
			resp.Entries = append(resp.Entries, summary(child))
		}
	}
	if it, ok := e.AsItem(); ok {
		for m := range it.Entries() {
			resp.Item = append(resp.Item, m.ID().String())
		}
	}
	for s := range e.Statements() {
		switch s := s.(type) {
		case library.Fetched:
			resp.Fetched = true
		case library.Link:
			resp.Links = append(resp.Links, linkResponseOf(id, s, ""))
		case library.Mapping:
			for _, k := range s.Links {
				resp.Links = append(resp.Links, linkResponseOf(id, k, s.Provider))
			}
		case library.Choice:
			resp.Chosen = &s.Chosen
		}
	}
	_ = writeJSON(w, http.StatusOK, map[string]any{"data": resp})
}

func linkResponseOf(id library.ID, k library.Link, provider library.Provider) linkResponse {
	resp := linkResponse{Other: k.To.String(), Kind: k.Kind.String(), Provider: string(provider)}
	if k.To == id {
		resp.Other = k.From.String()
		if k.Kind == library.Contains {
			resp.Kind = "part of"
		}
	}
	for _, w := range k.Why {
		resp.Why = append(resp.Why, string(w))
	}
	return resp
}

type editRequest struct {
	Op     string   `json:"op" validate:"required,oneof=value link choice add remove"`
	Field  string   `json:"field" validate:"required_if=Op value"`
	Value  string   `json:"value"`
	Other  string   `json:"other" validate:"required_if=Op link"`
	Kind   string   `json:"kind"`
	Why    []string `json:"why"`
	Chosen *bool    `json:"chosen"`
	Entry  string   `json:"entry" validate:"required_if=Op add,required_if=Op remove"`
}

var linkKinds = map[string]library.LinkKind{}

func init() {
	for _, k := range []library.LinkKind{library.Same, library.NotSame, library.Contains} {
		linkKinds[k.String()] = k
	}
}

func (c *LibraryController) Edit(w http.ResponseWriter, r *http.Request) {
	var req editRequest
	if err := decodeJSON(r, &req); err != nil {
		RespondError(w, r, err)
		return
	}
	id, err := entryID(chi.URLParam(r, "id"))
	if err != nil {
		RespondError(w, r, err)
		return
	}
	var statement library.Statement
	takeBack := false
	switch req.Op {
	case "value":
		statement, takeBack = library.Value{Entry: id, Field: library.Field(req.Field), Text: req.Value}, req.Value == ""
	case "link":
		k := library.Link{From: id}
		if k.To, err = entryID(req.Other); err != nil {
			break
		}
		if req.Kind != "" {
			var ok bool
			if k.Kind, ok = linkKinds[req.Kind]; !ok {
				err = errBadRequest
				break
			}
		}
		for _, w := range req.Why {
			k.Why = append(k.Why, library.Evidence(w))
		}
		statement, takeBack = k, req.Kind == ""
	case "choice":
		c := library.Choice{Entry: id}
		if req.Chosen != nil {
			c.Chosen = *req.Chosen
		}
		statement, takeBack = c, req.Chosen == nil
	case "add", "remove":
		var entry library.ID
		if entry, err = entryID(req.Entry); err != nil {
			break
		}
		statement, takeBack = library.Move{Entry: entry, Group: id}, req.Op == "remove"
	}
	if err == nil && takeBack {
		_, err = c.Library.Remove(r.Context(), statement)
	} else if err == nil {
		err = c.Library.Put(r.Context(), statement)
	}
	if err != nil {
		RespondError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type summaryResponse struct {
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

func summary(e library.Entry) summaryResponse {
	s := summaryResponse{ID: e.ID().String()}
	for _, text := range e.Values("title") {
		s.Title = text
		break
	}
	return s
}

type overviewResponse struct {
	Works   []summaryResponse `json:"works"`
	Reports []reportResponse  `json:"reports,omitempty"`
}

type reportResponse struct {
	Kind    string   `json:"kind"`
	Entries []string `json:"entries"`
}

type valueResponse struct {
	Text        string `json:"text"`
	Arrangement string `json:"arrangement,omitempty"`
	Edited      bool   `json:"edited,omitempty"`
}

type linkResponse struct {
	Other    string   `json:"other"`
	Kind     string   `json:"kind"`
	Why      []string `json:"why,omitempty"`
	Provider string   `json:"provider,omitempty"`
}

type entryResponse struct {
	ID      string                     `json:"id"`
	Fetched bool                       `json:"fetched"`
	Values  map[string][]valueResponse `json:"values"`
	Entries []summaryResponse          `json:"entries,omitempty"`
	Item    []string                   `json:"item,omitempty"`
	Links   []linkResponse             `json:"links,omitempty"`
	Chosen  *bool                      `json:"chosen,omitempty"`
}
