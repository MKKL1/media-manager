package anime_list_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
	"server/internal/plugin/wasm"
	anime_list "server/plugins/anime-list"
	"server/plugins/tvdb"
)

func TestRealSoloLeveling(t *testing.T) {
	tmdbKey, tvdbKey := os.Getenv("TMDB_API_KEY"), os.Getenv("TVDB_API_KEY")
	if tmdbKey == "" || tvdbKey == "" {
		t.Skip("TMDB_API_KEY and TVDB_API_KEY not set; skipping live test")
	}
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	must(t, err)
	_, err = plugin.Fetch(ctx, lib, tvdb.NewPlugin(tvdbKey, os.Getenv("TVDB_PIN")), "", "series:389597")
	must(t, err)
	tmdbWasm := filepath.Join(t.TempDir(), "tmdb.wasm")
	build := exec.Command("go", "build", "-buildmode=c-shared", "-o", tmdbWasm, ".")
	build.Dir, build.Env = "../wasm/tmdb", append(os.Environ(), "GOOS=wasip1", "GOARCH=wasm")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build tmdb: %v\n%s", err, b)
	}
	tmdb, err := wasm.Load(ctx, wasm.Config{Path: tmdbWasm, AllowedHosts: []string{"api.themoviedb.org"}, Config: map[string]string{"apikey": tmdbKey}})
	must(t, err)
	_, err = plugin.Fetch(ctx, lib, tmdb.Provider, "", "tv:127532")
	must(t, err)

	p := anime_list.NewPlugin("")
	refs, err := p.Refs(ctx, lib)
	must(t, err)
	t.Log("refs:", refs)
	for _, ref := range refs {
		must(t, plugin.FetchMapping(ctx, lib, p, ref))
	}

	season1Entry, _ := lib.Entry(library.ID{Provider: "tmdb", ProviderID: "tv:127532:s1"})
	season1, ok := season1Entry.AsGroup()
	if !ok {
		t.Fatal("TMDB season 1 not fetched")
	}
	episodes := 0
	for e := range season1.Entries() {
		episodes++
		item, _ := e.AsItem()
		linked := false
		for member := range item.Entries() {
			linked = linked || member.ID().Provider == "tvdb"
		}
		switch {
		case !linked && episodes <= 12:
			t.Errorf("%v has no TVDB episode in its item", e)
		case !linked:
			t.Logf("%v unlinked: the list doesn't place it", e)
		}
	}
	if episodes < 12 {
		t.Fatalf("TMDB season 1 has %d episodes, want at least 12", episodes)
	}
	for r := range lib.Reports() {
		if r.Kind == library.Disagreement {
			t.Errorf("disagreement: %v", r.Entries)
		}
	}
}
