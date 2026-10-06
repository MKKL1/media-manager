package tvdb

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
)

var fakeTVDB = map[string]string{
	"/series/1/extended": `{"data":{"id":1,"name":"Serie","originalLanguage":"jpn","status":{"name":"Ended"},
		"remoteIds":[{"id":"tt1","sourceName":"IMDB"},{"id":"77","sourceName":"TheMovieDB.com"}],
		"seasons":[{"number":1,"name":"First","type":{"type":"official"}}]}}`,
	"/series/1/translations/eng": `{"data":{"name":"Show","overview":"English"}}`,
	"/series/1/episodes/official/eng": `{"data":{"episodes":[
		{"id":100,"name":"Pilot","number":1,"seasonNumber":1}]},"links":{"next":"x?page=1"}}`,
	"/series/1/episodes/dvd/eng": `{"data":{"episodes":[
		{"id":101,"name":"Second","number":1,"seasonNumber":1},
		{"id":100,"name":"Pilot","number":2,"seasonNumber":1}]},"links":{"next":null}}`,
	"/series/2/extended":              `{"data":{"id":2,"name":"Other"}}`,
	"/series/2/episodes/official/eng": `{"data":{"episodes":[{"id":200,"name":"Only","number":1,"seasonNumber":1}]}}`,
	"/series/2/episodes/dvd/eng":      `{"data":{"episodes":[]}}`,
	"/movies/10/extended":             `{"data":{"id":10,"name":"Film","year":"1999","genres":[{"name":"Drama"}]}}`,
}

var fakeTVDBPage1 = `{"data":{"episodes":[{"id":101,"name":"Second","number":2,"seasonNumber":1}]},"links":{"next":null}}`

func TestPlugin(t *testing.T) {
	logins := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			logins++
			w.Write([]byte(`{"data":{"token":"t"}}`))
			return
		}
		if r.Header.Get("Authorization") != "Bearer t" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		body, ok := fakeTVDB[r.URL.Path]
		if r.URL.Path == "/series/1/episodes/official/eng" && r.URL.Query().Get("page") == "1" {
			body, ok = fakeTVDBPage1, true
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	defer server.Close()
	p := NewPlugin("key", "")
	p.client.baseURL = server.URL
	ctx := context.Background()
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}

	for _, arrangement := range []string{Aired, DVD} {
		if _, err := plugin.Fetch(ctx, lib, p, arrangement, "series:1"); err != nil {
			t.Fatal(arrangement, err)
		}
	}
	pilot, _ := lib.Entry(library.ID{Provider: Provider, ProviderID: "episode:100"})
	n := 0
	for range pilot.Values("title") {
		n++
	}
	if n != 2 {
		t.Errorf("the pilot has %d titles, want one per arrangement", n)
	}
	for _, c := range []struct{ season, want string }{
		{"series:1:s1", "episode:100"}, {"series:1:dvd:s1", "episode:101"},
	} {
		seasonEntry, _ := lib.Entry(library.ID{Provider: Provider, ProviderID: c.season})
		season, ok := seasonEntry.AsGroup()
		if !ok {
			t.Fatal("no season", c.season)
		}
		var order []string
		for e := range season.Entries() {
			order = append(order, e.ID().ProviderID)
		}
		if len(order) != 2 || order[0] != c.want {
			t.Errorf("%s order = %v, want %s first", c.season, order, c.want)
		}
	}

	if err := plugin.Refresh(ctx, lib, p, DVD); err != nil {
		t.Fatal(err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, Aired, "series:2"); err != nil {
		t.Error("aired-only series:", err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, DVD, "series:2"); !errors.Is(err, plugin.ErrNoSuchArrangement) {
		t.Errorf("series without a DVD order: %v", err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, "", "series:3"); !errors.Is(err, plugin.ErrNotFound) {
		t.Errorf("unknown series: %v", err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, "", "movie:10"); err != nil {
		t.Error("movie:", err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, DVD, "movie:10"); !errors.Is(err, plugin.ErrNoSuchArrangement) {
		t.Errorf("movie in DVD: %v", err)
	}
	if logins != 1 {
		t.Errorf("logged in %d times, want once", logins)
	}
}

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}
