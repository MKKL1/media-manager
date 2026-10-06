package anime_list_test

import (
	"context"
	"fmt"
	"iter"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
	anime_list "server/plugins/anime-list"
)

var ctx = context.Background()

const fakeList = `<anime-list>
  <anime anidbid="1" tvdbid="100" defaulttvdbseason="1" tmdbtv="200" tmdbseason="1">
    <name>Show</name>
    <mapping-list>
      <mapping anidbseason="0" tvdbseason="0">;1-1;</mapping>
      <mapping anidbseason="0" tmdbseason="0">;1-2;</mapping>
      <mapping anidbseason="1" tvdbseason="1">;3-3+4;</mapping>
    </mapping-list>
  </anime>
  <anime anidbid="2" tvdbid="100" defaulttvdbseason="2" tmdbtv="200" tmdbseason="1" tmdboffset="3">
    <name>Show 2nd cour</name>
  </anime>
  <anime anidbid="3" tvdbid="999" defaulttvdbseason="1" tmdbtv="200" tmdbseason="1"><name>Elsewhere</name></anime>
</anime-list>`

func episodes(provider library.Provider, idField, id string, seasons map[int]int) library.Fetched {
	top := library.Fetched{Provider: provider, Arrangement: "aired", Top: "show", CanHoldEntries: true,
		Values: map[library.Field]string{library.Field(idField): id}}
	for season := range 3 {
		count, ok := seasons[season]
		if !ok {
			continue
		}
		g := library.Tree{ProviderID: fmt.Sprint("s", season), CanHoldEntries: true}
		for n := 1; n <= count; n++ {
			g.Entries = append(g.Entries, library.Tree{
				ProviderID: fmt.Sprintf("%d:%d", season, n),
				Values:     map[library.Field]string{"season_number": fmt.Sprint(season), "episode_number": fmt.Sprint(n)},
			})
		}
		top.Entries = append(top.Entries, g)
	}
	return top
}

func TestLinksTVDBAndTMDBEpisodesTheListPlaces(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, fakeList) }))
	defer server.Close()
	p := anime_list.NewPlugin(server.URL)

	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	must(t, err)
	must(t, lib.Put(ctx, episodes("tvdb", "id.tvdb", "100", map[int]int{0: 1, 1: 4, 2: 2})))
	must(t, lib.Put(ctx, episodes("tmdb", "id.tmdb", "200", map[int]int{0: 2, 1: 5})))

	refs, err := p.Refs(ctx, lib)
	must(t, err)
	equal(t, refs, []string{"anidb:1", "anidb:2"})

	links := func(ref string) []string {
		got, err := p.FetchMapping(ctx, lib, ref)
		must(t, err)
		var out []string
		for _, k := range got.Links {
			out = append(out, fmt.Sprintf("%v %v %v", k.From, k.Kind, k.To))
		}
		return out
	}
	equal(t, links("anidb:1"), []string{
		"tvdb:0:1 same tmdb:0:2",
		"tvdb:1:1 same tmdb:1:1",
		"tvdb:1:2 same tmdb:1:2",
		"tmdb:1:3 contains tvdb:1:3",
		"tmdb:1:3 contains tvdb:1:4",
	})
	equal(t, links("anidb:2"), []string{"tvdb:2:1 same tmdb:1:4", "tvdb:2:2 same tmdb:1:5"})

	for _, ref := range refs {
		must(t, plugin.FetchMapping(ctx, lib, p, ref))
	}
	episode, _ := lib.Entry(library.ID{Provider: "tmdb", ProviderID: "1:4"})
	item, _ := episode.AsItem()
	var members []string
	for e := range item.Entries() {
		members = append(members, e.String())
	}
	slices.Sort(members)
	equal(t, members, []string{"tmdb:1:4", "tvdb:2:1"})

	if _, err := p.FetchMapping(ctx, lib, "anidb:404"); err == nil || !strings.Contains(err.Error(), "not listed") {
		t.Fatalf("an AniDB entry not listed: %v", err)
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func equal(t *testing.T, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got\n%q\nwant\n%q", got, want)
	}
}

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}
