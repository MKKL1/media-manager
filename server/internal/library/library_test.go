package library_test

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"sync"
	"testing"

	"server/internal/event"
	"server/internal/library"
)

const (
	tmdb   library.Provider = "tmdb"
	tvmaze library.Provider = "tvmaze"
	title  library.Field    = "title"
)

var ctx = context.Background()

func id(p library.Provider, s string) library.ID { return library.ID{Provider: p, ProviderID: s} }

func item(s, t string) library.Tree {
	return library.Tree{ProviderID: s, Values: map[library.Field]string{title: t}}
}

func group(s string, entries ...library.Tree) library.Tree {
	return library.Tree{ProviderID: s, CanHoldEntries: true, Entries: entries}
}

func fetched(p library.Provider, top library.Tree) library.Fetched {
	return fetchedIn(p, "aired", top)
}

func fetchedIn(p library.Provider, arrangement string, top library.Tree) library.Fetched {
	return library.Fetched{Provider: p, Arrangement: arrangement, Top: top.ProviderID, CanHoldEntries: top.CanHoldEntries, Values: top.Values, Entries: top.Entries}
}

type memStore struct {
	saved []savedStatement
	order int
}

type savedStatement struct {
	key                 string
	s                   library.Statement
	firstPut, lastSaved int
}

func key(s library.Statement) string {
	switch s := s.(type) {
	case library.Fetched:
		return fmt.Sprint("0 fetched ", s.Provider, " ", s.Arrangement, " ", s.Top)
	case library.Mapping:
		return fmt.Sprint("1 links ", s.Provider, " ", s.OwnID)
	case library.Value:
		return fmt.Sprint("2 value ", s.Entry, " ", s.Field)
	case library.Move:
		return fmt.Sprint("2 move ", s.Entry, " ", s.Group)
	case library.Link:
		a, b := s.From.String(), s.To.String()
		return fmt.Sprint("2 link ", min(a, b), " ", max(a, b))
	case library.Choice:
		return fmt.Sprint("2 choice ", s.Entry)
	}
	panic(s)
}

func (m *memStore) Put(_ context.Context, s library.Statement) error {
	m.order++
	k := key(s)
	for i := range m.saved {
		if m.saved[i].key == k {
			m.saved[i].s, m.saved[i].lastSaved = s, m.order
			return nil
		}
	}
	m.saved = append(m.saved, savedStatement{k, s, m.order, m.order})
	return nil
}

func (m *memStore) Remove(_ context.Context, s library.Statement) error {
	k := key(s)
	m.saved = slices.DeleteFunc(m.saved, func(x savedStatement) bool { return x.key == k })
	return nil
}

func (m *memStore) All(context.Context) iter.Seq2[library.Statement, error] {
	sorted := slices.Clone(m.saved)
	slices.SortFunc(sorted, func(x, y savedStatement) int {
		if c := cmp.Compare(x.key[0], y.key[0]); c != 0 {
			return c
		}
		if x.key[0] == '2' {
			return cmp.Compare(x.lastSaved, y.lastSaved)
		}
		return cmp.Compare(x.firstPut, y.firstPut)
	})
	return func(yield func(library.Statement, error) bool) {
		for _, x := range sorted {
			if !yield(x.s, nil) {
				return
			}
		}
	}
}

func open(t testing.TB) (*library.Library, *memStore) {
	t.Helper()
	s := &memStore{}
	l, err := library.Open(ctx, s, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	return l, s
}

func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func put(t testing.TB, l *library.Library, statements ...library.Statement) {
	t.Helper()
	for _, s := range statements {
		must(t, l.Put(ctx, s))
	}
}

func remove(t testing.TB, l *library.Library, s library.Statement) {
	t.Helper()
	removed, err := l.Remove(ctx, s)
	must(t, err)
	if !removed {
		t.Fatalf("%+v was not removed", s)
	}
}

func entry(t testing.TB, l *library.Library, e library.ID) library.Entry {
	t.Helper()
	got, ok := l.Entry(e)
	if !ok {
		t.Fatalf("no statement names %v", e)
	}
	return got
}

func groupEntries(t testing.TB, l *library.Library, g library.ID) []string {
	t.Helper()
	gr, ok := entry(t, l, g).AsGroup()
	if !ok {
		t.Fatalf("%v is not a group", g)
	}
	return ids(gr.Entries())
}

func ids(seq iter.Seq[library.Entry]) []string {
	var out []string
	for e := range seq {
		out = append(out, e.ID().String())
	}
	return out
}

func itemEntries(t testing.TB, l *library.Library, e library.ID) []string {
	t.Helper()
	i, ok := entry(t, l, e).AsItem()
	if !ok {
		t.Fatalf("%v has no item", e)
	}
	return ids(i.Entries())
}

func reports(l *library.Library, kind library.ReportKind) []string {
	var out []string
	for r := range l.Reports() {
		if r.Kind == kind {
			out = append(out, strings.Join(ids(slices.Values(r.Entries)), " "))
		}
	}
	return out
}

func equal(t testing.TB, got, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEveryValueIsKeptWithTheStatementItCameFrom(t *testing.T) {
	l, _ := open(t)
	put(t, l,
		fetched(tmdb, group("show", item("ep1", "The Heist"))),
		fetchedIn(tmdb, "dvd", group("dvd-show", item("ep1", "The Heist (Extended)"))),
		library.Value{Entry: id(tmdb, "ep1"), Field: title, Text: "Mine"})

	var got []string
	for s, text := range entry(t, l, id(tmdb, "ep1")).Values(title) {
		switch s := s.(type) {
		case library.Value:
			got = append(got, text+"|edit")
		case library.Fetched:
			got = append(got, text+"|"+s.Arrangement+" "+s.Top)
		}
	}
	equal(t, got, []string{"Mine|edit", "The Heist|aired show", "The Heist (Extended)|dvd dvd-show"})

	remove(t, l, library.Value{Entry: id(tmdb, "ep1"), Field: title})
	n := 0
	for range entry(t, l, id(tmdb, "ep1")).Values(title) {
		n++
	}
	if n != 2 {
		t.Fatalf("after Remove: %d values, want 2", n)
	}
	if removed, err := l.Remove(ctx, library.Value{Entry: id(tmdb, "ep1"), Field: title}); removed || err != nil {
		t.Fatalf("removing again: %v %v, want false and no error", removed, err)
	}
}

func TestSameIsPassedAlongAndRemovingTheLinkSplits(t *testing.T) {
	l, _ := open(t)
	put(t, l,
		fetched(tmdb, group("show", item("1", "Pilot"))),
		fetched(tvmaze, group("show", item("1", "Pilot"))),
		fetched("anilist", item("1", "Episode 1")),
		library.Link{From: id(tmdb, "1"), To: id(tvmaze, "1"), Kind: library.Same, Why: []library.Evidence{"air date", "title"}},
		library.Link{From: id(tvmaze, "1"), To: id("anilist", "1"), Kind: library.Same})

	equal(t, itemEntries(t, l, id("anilist", "1")), []string{"tmdb:1", "tvmaze:1", "anilist:1"})

	remove(t, l, library.Link{From: id("anilist", "1"), To: id(tvmaze, "1")})
	equal(t, itemEntries(t, l, id(tmdb, "1")), []string{"tmdb:1", "tvmaze:1"})
	equal(t, itemEntries(t, l, id("anilist", "1")), []string{"anilist:1"})
}

func TestNotSameInsideAnItemIsAConflict(t *testing.T) {
	l, _ := open(t)
	for _, p := range []library.Provider{tmdb, tvmaze, "anilist"} {
		put(t, l, fetched(p, item("1", "Pilot")))
	}
	put(t, l,
		library.Link{From: id(tmdb, "1"), To: id(tvmaze, "1"), Kind: library.Same},
		library.Link{From: id(tvmaze, "1"), To: id("anilist", "1"), Kind: library.Same},
		library.Link{From: id("anilist", "1"), To: id(tmdb, "1"), Kind: library.NotSame})

	equal(t, reports(l, library.Conflict), []string{"tmdb:1 anilist:1"})

	var links []string
	for s := range entry(t, l, id(tmdb, "1")).Statements() {
		if k, ok := s.(library.Link); ok {
			links = append(links, k.From.String()+" "+k.Kind.String()+" "+k.To.String())
		}
	}
	equal(t, links, []string{"tmdb:1 same tvmaze:1", "tmdb:1 not same anilist:1"})
}

func TestContainsWaitsForADecision(t *testing.T) {
	l, _ := open(t)
	put(t, l,
		fetched(tmdb, group("s1", item("e1", "Pilot Part 1"), item("e2", "Pilot Part 2"))),
		fetched(tvmaze, group("s1", item("pilot", "Pilot"))),
		library.Link{From: id(tvmaze, "pilot"), To: id(tmdb, "e1"), Kind: library.Contains},
		library.Link{From: id(tvmaze, "pilot"), To: id(tmdb, "e2"), Kind: library.Contains})

	if got := reports(l, library.Undecided); !slices.Contains(got, "tvmaze:pilot") {
		t.Fatalf("undecided %q, want tvmaze:pilot", got)
	}
	put(t, l, library.Choice{Entry: id(tvmaze, "pilot"), Chosen: true})
	if got := reports(l, library.Undecided); slices.Contains(got, "tvmaze:pilot") {
		t.Fatalf("still undecided after a choice: %q", got)
	}
	put(t, l, library.Choice{Entry: id(tmdb, "e2"), Chosen: true})
	if n := len(reports(l, library.Conflict)); n != 1 {
		t.Fatalf("%d conflicts, want 1 (whole and part both chosen)", n)
	}
	remove(t, l, library.Choice{Entry: id(tmdb, "e2")})
	if n := len(reports(l, library.Conflict)); n != 0 {
		t.Fatalf("%d conflicts after removing the choice, want 0", n)
	}
}

func TestTopLevelEntriesHoldingTheSameItemsWaitForADecision(t *testing.T) {
	l, _ := open(t)
	put(t, l,
		fetched(tmdb, group("show", group("s1", item("1", "One"), item("special", "Special")))),
		fetched(tvmaze, group("show", group("s1", item("1", "One")))),
		fetched(tmdb, item("movie", "A movie")),
		library.Link{From: id(tmdb, "1"), To: id(tvmaze, "1"), Kind: library.Same})

	equal(t, ids(l.Works()), []string{"tmdb:movie"})
	equal(t, reports(l, library.Undecided), []string{"tmdb:show", "tvmaze:show"})
	equal(t, reports(l, library.Unplaced), nil)

	put(t, l, library.Choice{Entry: id(tvmaze, "show"), Chosen: true})
	equal(t, ids(l.Works()), []string{"tvmaze:show", "tmdb:movie"})
	equal(t, reports(l, library.Undecided), nil)
	equal(t, reports(l, library.Unplaced), []string{"tmdb:special"})

	put(t, l, library.Choice{Entry: id(tmdb, "show"), Chosen: true})
	if n := len(reports(l, library.Conflict)); n != 1 {
		t.Fatalf("%d conflicts, want 1 (two chosen top-level entries)", n)
	}
}

func TestMoveAndRemoveTheMove(t *testing.T) {
	l, _ := open(t)
	put(t, l, fetched(tmdb, group("show",
		group("s0", item("special", "Special")),
		group("s2", item("e1", "One"), item("e2", "Two")))))

	put(t, l, library.Move{Entry: id(tmdb, "special"), Group: id(tmdb, "s2")})
	equal(t, groupEntries(t, l, id(tmdb, "s2")), []string{"tmdb:e1", "tmdb:e2", "tmdb:special"})
	equal(t, groupEntries(t, l, id(tmdb, "s0")), nil)

	if removed, err := l.Remove(ctx, library.Move{Entry: id(tmdb, "e1"), Group: id(tmdb, "s2")}); removed || err != nil {
		t.Fatalf("removing a move never made: %v %v, want false and no error", removed, err)
	}
	remove(t, l, library.Move{Entry: id(tmdb, "special"), Group: id(tmdb, "s2")})
	equal(t, groupEntries(t, l, id(tmdb, "s0")), []string{"tmdb:special"})

	if err := l.Put(ctx, library.Move{Entry: id(tmdb, "show"), Group: id(tmdb, "s2")}); !errors.Is(err, library.ErrRefused) {
		t.Fatalf("moving a group under its own child: %v, want refused", err)
	}
}

func TestStatementsThatBreakARuleAreRefused(t *testing.T) {
	l, _ := open(t)
	put(t, l, fetched(tmdb, group("s1", item("e1", "One"))), fetched(tvmaze, group("s1", item("e1", "One"))))
	s1, e1, other := id(tmdb, "s1"), id(tmdb, "e1"), id(tvmaze, "e1")
	for name, s := range map[string]library.Statement{
		"an item with entries":            fetched(tmdb, library.Tree{ProviderID: "x", Entries: []library.Tree{item("y", "")}}),
		"a fetch without an arrangement":  fetchedIn(tmdb, "", item("x", "")),
		"linking a group":                 library.Link{From: s1, To: e1, Kind: library.Same},
		"linking an entry to itself":      library.Link{From: e1, To: e1, Kind: library.Same},
		"a kind that isn't one":           library.Link{From: e1, To: other, Kind: 0},
		"an empty value":                  library.Value{Entry: e1, Field: title},
		"moving another provider's entry": library.Move{Entry: other, Group: s1},
		"moving a group into itself":      library.Move{Entry: s1, Group: s1},
		"an id without a provider":        library.Choice{Entry: library.ID{ProviderID: "x"}, Chosen: true},
		"no statement":                    nil,
	} {
		if err := l.Put(ctx, s); !errors.Is(err, library.ErrRefused) {
			t.Errorf("%s: %v, want refused", name, err)
		}
	}
}

func TestEditsOutliveTheirEntries(t *testing.T) {
	l, _ := open(t)
	put(t, l, fetched(tmdb, group("show", item("e1", "One"))), library.Value{Entry: id(tmdb, "e1"), Field: title, Text: "Mine"})
	remove(t, l, library.Fetched{Provider: tmdb, Arrangement: "aired", Top: "show"})

	equal(t, reports(l, library.UnappliedEdit), []string{"tmdb:e1"})
	if _, ok := entry(t, l, id(tmdb, "e1")).AsItem(); ok {
		t.Fatal("removed entry still an item")
	}
	if _, ok := l.Entry(id(tmdb, "show")); ok {
		t.Fatal("no statement names tmdb:show any more, but Entry finds it")
	}
	put(t, l, fetched(tmdb, group("show", item("e1", "One"))))
	equal(t, reports(l, library.UnappliedEdit), nil)
	for s := range entry(t, l, id(tmdb, "e1")).Values(title) {
		if _, ok := s.(library.Value); !ok {
			t.Fatalf("edit didn't come back first: %+v", s)
		}
		break
	}
}

func TestOpenLoadsTheSameLibrary(t *testing.T) {
	l, s := open(t)
	put(t, l,
		fetched(tmdb, group("show", group("s0", item("special", "Special")), group("s1", item("1", "One")))),
		fetched(tvmaze, item("1", "One")),
		library.Link{From: id(tmdb, "1"), To: id(tvmaze, "1"), Kind: library.Same, Why: []library.Evidence{"title"}},
		library.Value{Entry: id(tmdb, "1"), Field: title, Text: "Mine"},
		library.Move{Entry: id(tmdb, "special"), Group: id(tmdb, "s1")},
		library.Choice{Entry: id(tmdb, "show"), Chosen: true})

	again, err := library.Open(ctx, s, &event.Bus{})
	must(t, err)
	for _, lib := range []*library.Library{l, again} {
		equal(t, ids(lib.Works()), []string{"tmdb:show"})
		equal(t, itemEntries(t, lib, id(tvmaze, "1")), []string{"tmdb:1", "tvmaze:1"})
		equal(t, groupEntries(t, lib, id(tmdb, "s1")), []string{"tmdb:1", "tmdb:special"})
	}
}

func TestReadsDuringWrites(t *testing.T) {
	l, _ := open(t)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := range 200 {
			put(t, l,
				fetched(tmdb, group(fmt.Sprint("show", i%7), item(fmt.Sprint("e", i), "x"))),
				library.Link{From: id(tmdb, fmt.Sprint("e", i)), To: id(tvmaze, fmt.Sprint("e", i)), Kind: library.Same})
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			for top := range l.Works() {
				if g, ok := top.AsGroup(); ok {
					for e := range g.Entries() {
						e.AsItem()
					}
				}
			}
		}
	}()
	wg.Wait()
}

func TestMappingsCountWhenTheyAgree(t *testing.T) {
	l, s := open(t)
	put(t, l,
		fetched(tmdb, group("show", item("1", "One"), item("2", "Two"))),
		fetched(tvmaze, group("show", item("1", "One"), item("2", "Two"))))
	same := func(a, b string) library.Link {
		return library.Link{From: id(tmdb, a), To: id(tvmaze, b), Kind: library.Same, Why: []library.Evidence{"list"}}
	}
	put(t, l,
		library.Mapping{Provider: "lists", OwnID: "show", Links: []library.Link{same("1", "1"), same("2", "2")}},
		library.Mapping{Provider: "other", OwnID: "x", Links: []library.Link{same("1", "1")}})
	equal(t, itemEntries(t, l, id(tmdb, "1")), []string{"tmdb:1", "tvmaze:1"})
	equal(t, itemEntries(t, l, id(tmdb, "2")), []string{"tmdb:2", "tvmaze:2"})

	put(t, l, library.Mapping{Provider: "other", OwnID: "x", Links: []library.Link{
		same("1", "1"), {From: id(tvmaze, "2"), To: id(tmdb, "2"), Kind: library.NotSame}}})
	equal(t, itemEntries(t, l, id(tmdb, "2")), []string{"tmdb:2"})
	equal(t, reports(l, library.Disagreement), []string{"tmdb:2 tvmaze:2"})

	put(t, l, library.Link{From: id(tmdb, "2"), To: id(tvmaze, "2"), Kind: library.Same, Why: []library.Evidence{"a person"}})
	equal(t, itemEntries(t, l, id(tmdb, "2")), []string{"tmdb:2", "tvmaze:2"})
	equal(t, reports(l, library.Disagreement), nil)
	var got []string
	for s := range entry(t, l, id(tmdb, "2")).Statements() {
		switch s := s.(type) {
		case library.Link:
			got = append(got, "edit|"+s.Kind.String())
		case library.Mapping:
			got = append(got, string(s.Provider)+"|"+s.Links[0].Kind.String())
		}
	}
	equal(t, got, []string{"edit|same", "lists|same", "other|not same"})

	remove(t, l, library.Mapping{Provider: "lists", OwnID: "show"})
	equal(t, itemEntries(t, l, id(tmdb, "1")), []string{"tmdb:1", "tvmaze:1"})

	reopened, err := library.Open(ctx, s, &event.Bus{})
	must(t, err)
	equal(t, itemEntries(t, reopened, id(tmdb, "1")), []string{"tmdb:1", "tvmaze:1"})
	equal(t, itemEntries(t, reopened, id(tmdb, "2")), []string{"tmdb:2", "tvmaze:2"})

	err = l.Put(ctx, library.Mapping{Provider: "lists", OwnID: "s", Links: []library.Link{{From: id(tmdb, "show"), To: id(tvmaze, "show"), Kind: library.Same}}})
	if !errors.Is(err, library.ErrRefused) {
		t.Fatalf("linking groups: %v, want refused", err)
	}
}

func TestArrangementsAreTheOnesFetchedIn(t *testing.T) {
	l, _ := open(t)
	put(t, l, fetchedIn(tmdb, "dvd", item("a", "A")), fetched(tmdb, item("b", "B")), fetched(tvmaze, item("c", "C")))
	var names []string
	for a := range l.Arrangements(tmdb) {
		names = append(names, a.Name())
	}
	equal(t, names, []string{"dvd", "aired"})

	remove(t, l, library.Fetched{Provider: tmdb, Arrangement: "dvd", Top: "a"})
	names = nil
	for a := range l.Arrangements(tmdb) {
		names = append(names, a.Name()+":"+strings.Join(ids(a.Top()), ","))
	}
	equal(t, names, []string{"aired:tmdb:b"})
}
