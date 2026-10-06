package wasm

import (
	"context"
	"errors"
	"iter"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
)

func buildTMDB(t *testing.T) string {
	t.Helper()
	out := filepath.Join(t.TempDir(), "tmdb.wasm")
	build := exec.Command("go", "build", "-buildmode=c-shared", "-o", out, ".")
	build.Dir = "../../../plugins/wasm/tmdb"
	build.Env = append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, b)
	}
	return out
}

var fakeTMDB = map[string]string{
	"/3/tv/1": `{"id":1,"name":"Show","external_ids":{"imdb_id":"tt1"},
		"seasons":[{"season_number":1}],
		"episode_groups":{"results":[{"id":"g","type":3}]}}`,
	"/3/tv/1/season/1": `{"season_number":1,"episodes":[
		{"season_number":1,"episode_number":1,"name":"Pilot"},
		{"season_number":1,"episode_number":2,"name":"Second"}]}`,
	"/3/tv/episode_group/g": `{"id":"g","name":"DVD","groups":[{"id":"d1","name":"Disc 1","order":1,"episodes":[
		{"season_number":1,"episode_number":2,"name":"Second"},
		{"season_number":1,"episode_number":1,"name":"Pilot"}]}]}`,
	"/3/tv/2":         `{"id":2,"name":"Other","original_language":"ja","genres":[{"id":16,"name":"Animation"}],"seasons":[]}`,
	"/3/search/multi": `{"results":[{"id":1,"media_type":"tv","name":"Show"},{"id":9,"media_type":"person","name":"Someone"}]}`,
}

func TestTMDBThroughExtism(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := fakeTMDB[r.URL.Path]
		if !ok || r.URL.Query().Get("api_key") != "0123456789abcdef0123456789abcdef" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(body))
	}))
	defer server.Close()
	ctx := context.Background()
	path := buildTMDB(t)
	loaded, err := Load(ctx, Config{
		Path:         path,
		AllowedHosts: []string{"127.0.0.1"},
		Config:       map[string]string{"apikey": "0123456789abcdef0123456789abcdef", "base_url": server.URL},
	})
	if err != nil {
		t.Fatal(err)
	}
	p := loaded.Provider
	if loaded.Type != nil {
		t.Error("tmdb is no metadata type")
	}
	if p.Provider() != "tmdb" || plugin.ArrangementsOf(p)[0] != "aired" || len(plugin.TagsOf(p)) != 3 {
		t.Errorf("info: %v %v %v", p.Provider(), plugin.ArrangementsOf(p), plugin.TagsOf(p))
	}
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}

	for _, arrangement := range []string{"aired", "dvd"} {
		if _, err := plugin.Fetch(ctx, lib, p, arrangement, "tv:1"); err != nil {
			t.Fatal(arrangement, err)
		}
	}
	pilot, _ := lib.Entry(library.ID{Provider: "tmdb", ProviderID: "tv:1:s1:e1"})
	n := 0
	for range pilot.Values("title") {
		n++
	}
	if n != 2 {
		t.Errorf("the pilot has %d titles, want one per arrangement", n)
	}
	discEntry, _ := lib.Entry(library.ID{Provider: "tmdb", ProviderID: "tv:1:group:d1"})
	disc, ok := discEntry.AsGroup()
	if !ok {
		t.Fatal("no DVD group")
	}
	var order []string
	for e := range disc.Entries() {
		order = append(order, e.ID().ProviderID)
	}
	if len(order) != 2 || order[0] != "tv:1:s1:e2" {
		t.Errorf("DVD order = %v", order)
	}
	if err := plugin.Refresh(ctx, lib, p, "dvd"); err != nil {
		t.Fatal(err)
	}

	if _, err := plugin.Fetch(ctx, lib, p, "dvd", "tv:2"); !errors.Is(err, plugin.ErrNoSuchArrangement) {
		t.Errorf("show without a DVD order: %v", err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, "", "tv:3"); !errors.Is(err, plugin.ErrNotFound) {
		t.Errorf("unknown show: %v", err)
	}
	if _, err := plugin.Fetch(ctx, lib, p, "", "tv:2"); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ id, tags string }{{"tv:1", "default:tv"}, {"tv:2", "default:tv default:anime"}} {
		got := ""
		work, _ := lib.Entry(library.ID{Provider: "tmdb", ProviderID: c.id})
		for _, text := range work.Values(plugin.TagsField) {
			got = text
		}
		if got != c.tags {
			t.Errorf("%s tags = %q, want %q", c.id, got, c.tags)
		}
	}

	found, err := plugin.Search(ctx, p, "show")
	if err != nil || len(found) != 1 || found[0].Ref != "tv:1" || found[0].Values[plugin.TagsField] != "default:tv" {
		t.Errorf("search = %v, %v", found, err)
	}

	offline, err := Load(ctx, Config{Path: path, Config: map[string]string{"base_url": server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := offline.Provider.Fetch(ctx, "aired", "tv:1"); err == nil || errors.Is(err, plugin.ErrNotFound) {
		t.Errorf("a plugin reached a host it isn't allowed: %v", err)
	}
}

func TestInfoAndTagsChecked(t *testing.T) {
	for _, c := range []struct {
		info info
		ok   bool
	}{
		{info{Provider: "tmdb", Tags: []plugin.Tag{"default:tv", "tmdb:special"}}, true},
		{info{Provider: "tmdb", Tags: []plugin.Tag{"anilist:anime"}}, false},
		{info{Provider: "tmdb", Tags: []plugin.Tag{"tv"}}, false},
		{info{Provider: "TMDB"}, false},
		{info{Provider: ""}, false},
		{info{Formats: []plugin.Tag{"default:tv"}}, true},
		{info{Formats: []plugin.Tag{"tv"}}, false},
	} {
		if err := c.info.check(); (err == nil) != c.ok {
			t.Errorf("%+v: %v", c.info, err)
		}
	}

	p := &Plugin{info: info{Provider: "tmdb", Tags: []plugin.Tag{"default:tv"}}}
	if _, err := p.fetched(entry{ID: "tv:1", Tags: []plugin.Tag{"default:movie"}}); err == nil {
		t.Error("a tag the plugin didn't declare")
	}
	if _, err := p.fetched(entry{ID: "tv:1", Values: map[library.Field]string{plugin.TagsField: "default:tv"}}); err == nil {
		t.Error("tags set as a value")
	}
	if _, err := p.fetched(entry{ID: "tv:1", CanHoldEntries: true, Entries: []entry{{}}}); err == nil {
		t.Error("an entry without an id")
	}
}

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}
