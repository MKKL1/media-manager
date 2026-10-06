package http

import (
	"context"
	"encoding/json"
	"iter"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rs/zerolog"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
)

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}

type fakePlugin struct{ provider library.Provider }

func (p fakePlugin) Provider() library.Provider { return p.provider }

type animePlugin struct{ fakePlugin }

func (animePlugin) Tags() []plugin.Tag { return []plugin.Tag{"default:tv", "default:anime"} }
func (p fakePlugin) Fetch(_ context.Context, _, ref string) (library.Tree, error) {
	return library.Tree{
		ProviderID: ref, CanHoldEntries: true,
		Values: map[library.Field]string{"title": "Show"},
		Entries: []library.Tree{{
			ProviderID: ref + ":e1",
			Values:     map[library.Field]string{"title": "Pilot"},
		}},
	}, nil
}

func TestEditsThroughHTTP(t *testing.T) {
	lib, err := library.Open(context.Background(), nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	r := NewRouter(zerolog.Nop(), anyToken{}, allowAll{})
	(&LibraryController{Library: lib}).RegisterRoutes(r)
	(&ProviderController{Library: lib, Plugins: func() []plugin.Plugin { return []plugin.Plugin{fakePlugin{"a"}, animePlugin{fakePlugin{"b"}}} }}).RegisterRoutes(r)

	call := func(method, path, body string, want int) map[string]any {
		t.Helper()
		w := httptest.NewRecorder()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer t")
		r.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: status %d, want %d: %s", method, path, w.Code, want, w.Body)
		}
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		out, _ = out["data"].(map[string]any)
		return out
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/providers?tag=default:anime", nil)
	req.Header.Set("Authorization", "Bearer t")
	r.ServeHTTP(w, req)
	if want := `{"data":[{"provider":"b","arrangements":["default"],"can_search":false,"tags":["default:tv","default:anime"]}]}`; strings.TrimSpace(w.Body.String()) != want {
		t.Errorf("providers tagged anime: %s", w.Body)
	}
	call("POST", "/api/v1/providers/a/fetch", `{"ref":"1"}`, 200)
	call("POST", "/api/v1/providers/b/fetch", `{"ref":"1"}`, 200)
	call("POST", "/api/v1/providers/nope/fetch", `{"ref":"1"}`, 404)
	call("GET", "/api/v1/providers/a/search?q=show", "", 501)
	call("POST", "/api/v1/entries/a:1:e1/edits", `{"op":"value","field":"title","value":"Episode One"}`, 204)
	call("POST", "/api/v1/entries/a:1:e1/edits", `{"op":"link","other":"b:1:e1","kind":"same","why":["same title"]}`, 204)
	call("POST", "/api/v1/entries/a:1:e1/edits", `{"op":"link","other":"b:1:e1","kind":"bogus"}`, 400)
	call("POST", "/api/v1/entries/a:1/edits", `{"op":"link","other":"b:1","kind":"same"}`, 409)
	call("POST", "/api/v1/entries/a:1/edits", `{"op":"bogus"}`, 422)

	if got := call("GET", "/api/v1/library", "", 200); len(got["reports"].([]any)) != 2 {
		t.Fatalf("want two undecided: %v", got)
	}
	call("POST", "/api/v1/entries/a:1/edits", `{"op":"choice","chosen":true}`, 204)
	works := call("GET", "/api/v1/library", "", 200)["works"].([]any)
	if len(works) != 1 || works[0].(map[string]any)["id"] != "a:1" {
		t.Fatalf("the work should be a:1: %v", works)
	}

	ep := call("GET", "/api/v1/entries/a:1:e1", "", 200)
	if titles := ep["values"].(map[string]any)["title"].([]any); titles[0].(map[string]any)["text"] != "Episode One" {
		t.Fatalf("the edit comes first: %v", titles)
	}
	if item := ep["item"].([]any); len(item) != 2 {
		t.Fatalf("item should join both episodes: %v", item)
	}
	call("GET", "/api/v1/entries/a:404", "", 404)
	call("POST", "/api/v1/entries/a:1:e1/edits", `{"op":"link","other":"b:1:e1"}`, 204)
	if item := call("GET", "/api/v1/entries/a:1:e1", "", 200)["item"].([]any); len(item) != 1 {
		t.Fatalf("taking the link back splits the item: %v", item)
	}
}
