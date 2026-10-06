package tvdb

import (
	"context"
	"os"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
)

func TestRealTVDB(t *testing.T) {
	key := os.Getenv("TVDB_API_KEY")
	if key == "" {
		t.Skip("TVDB_API_KEY not set; skipping live TVDB test")
	}
	ctx := context.Background()
	p := NewPlugin(key, os.Getenv("TVDB_PIN"))
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	for _, arrangement := range []string{Aired, DVD} {
		if _, err := plugin.Fetch(ctx, lib, p, arrangement, "series:78874"); err != nil {
			t.Fatal(arrangement, err)
		}
	}
	episodes := func(arrangement string) (all []library.Entry) {
		for a := range lib.Arrangements(Provider) {
			if a.Name() != arrangement {
				continue
			}
			for top := range a.Top() {
				show, _ := top.AsGroup()
				for s := range show.Entries() {
					season, _ := s.AsGroup()
					if isSpecials(s) {
						continue
					}
					for e := range season.Entries() {
						all = append(all, e)
					}
				}
			}
		}
		return all
	}
	if n := len(episodes(Aired)); n != 14 {
		t.Errorf("aired has %d episodes outside specials, want 14", n)
	}
	dvd := episodes(DVD)
	if len(dvd) == 0 {
		t.Error("dvd has no episodes")
	}
	shared := false
	for _, e := range dvd {
		n := 0
		for range e.Values("title") {
			n++
		}
		shared = shared || n == 2
	}
	if !shared {
		t.Error("no episode is the same entry in aired and dvd")
	}
	if _, err := plugin.Fetch(ctx, lib, p, Aired, "movie:169"); err != nil {
		t.Error("The Matrix:", err)
	}
}

func isSpecials(season library.Entry) bool {
	for _, text := range season.Values("season_number") {
		if text == "0" {
			return true
		}
	}
	return false
}
