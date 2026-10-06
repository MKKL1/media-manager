package plugin_test

import (
	"context"
	"errors"
	"iter"
	"maps"
	"slices"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
)

var ctx = context.Background()

type fake map[string]library.Tree

func (fake) Provider() library.Provider { return "fake" }
func (fake) Arrangements() []string     { return []string{"aired", "dvd"} }
func (f fake) Fetch(_ context.Context, arrangement, ref string) (library.Tree, error) {
	if arrangement != "aired" {
		return library.Tree{}, plugin.ErrNoSuchArrangement
	}
	top, ok := f[ref]
	if !ok {
		return library.Tree{}, plugin.ErrNotFound
	}
	return top, nil
}

func show(ref string, episodes ...string) library.Tree {
	top := library.Tree{ProviderID: ref, CanHoldEntries: true}
	for _, e := range episodes {
		top.Entries = append(top.Entries, library.Tree{ProviderID: e})
	}
	return top
}

func topIDs(lib *library.Library) []string {
	var ids []string
	for a := range lib.Arrangements("fake") {
		for e := range a.Top() {
			ids = append(ids, a.Name()+":"+e.ID().ProviderID)
		}
	}
	return ids
}

func isItem(lib *library.Library, id string) bool {
	e, ok := lib.Entry(library.ID{Provider: "fake", ProviderID: id})
	if !ok {
		return false
	}
	_, ok = e.AsItem()
	return ok
}

func TestFetchAndRefresh(t *testing.T) {
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	p := fake{"a": show("a", "a1", "a2"), "b": show("b", "b1")}

	for _, ref := range slices.Sorted(maps.Keys(p)) {
		if _, err := plugin.Fetch(ctx, lib, p, "", ref); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := plugin.Fetch(ctx, lib, p, "absolute", "a"); !errors.Is(err, plugin.ErrNoSuchArrangement) {
		t.Fatalf("an arrangement not offered: got %v", err)
	}
	if got := topIDs(lib); !slices.Equal(got, []string{"aired:a", "aired:b"}) {
		t.Fatalf("top = %v", got)
	}

	delete(p, "b")
	p["a"] = show("a", "a1", "a2", "a3")
	if err := plugin.Refresh(ctx, lib, p, ""); err != nil {
		t.Fatal(err)
	}
	if got := topIDs(lib); !slices.Equal(got, []string{"aired:a"}) {
		t.Fatalf("after refresh top = %v", got)
	}
	if !isItem(lib, "a3") {
		t.Fatal("a3 not fetched")
	}
	if isItem(lib, "b1") {
		t.Fatal("b1 still fetched")
	}
}

type fakeMapping map[string][]library.Link

func (fakeMapping) Provider() library.Provider { return "fake-links" }
func (f fakeMapping) FetchMapping(_ context.Context, _ *library.Library, ref string) (library.Mapping, error) {
	links, ok := f[ref]
	if !ok {
		return library.Mapping{}, plugin.ErrNotFound
	}
	return library.Mapping{Provider: "fake-links", OwnID: ref, Links: links}, nil
}

func TestFetchMapping(t *testing.T) {
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	x, y := library.ID{Provider: "x", ProviderID: "1"}, library.ID{Provider: "y", ProviderID: "1"}
	p := fakeMapping{"a": {{From: x, To: y, Kind: library.Same}}}
	if err := plugin.FetchMapping(ctx, lib, p, "a"); err != nil {
		t.Fatal(err)
	}
	if err := plugin.FetchMapping(ctx, lib, p, "b"); !errors.Is(err, plugin.ErrNotFound) {
		t.Fatalf("a ref the provider doesn't have: got %v", err)
	}
	e, ok := lib.Entry(x)
	if !ok {
		t.Fatal("no statement names x:1")
	}
	for s := range e.Statements() {
		if links, ok := s.(library.Mapping); !ok || links.OwnID != "a" {
			t.Fatalf("statement %+v, want the mapping under a", s)
		}
	}
}

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}

type bare struct{}

func (bare) Provider() library.Provider { return "bare" }
func (bare) Fetch(_ context.Context, arrangement, ref string) (library.Tree, error) {
	return library.Tree{ProviderID: ref + "@" + arrangement}, nil
}

func TestPluginWithoutOptionalParts(t *testing.T) {
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := plugin.Fetch(ctx, lib, bare{}, "", "x")
	if err != nil || id.ProviderID != "x@"+plugin.Default {
		t.Fatalf("fetch in the default arrangement: %v %v", id, err)
	}
	if _, err := plugin.Fetch(ctx, lib, bare{}, "dvd", "x"); !errors.Is(err, plugin.ErrNoSuchArrangement) {
		t.Fatalf("only Default is offered: got %v", err)
	}
	if _, err := plugin.Search(ctx, bare{}, "x"); !errors.Is(err, errors.ErrUnsupported) {
		t.Fatalf("no Searcher part: got %v", err)
	}
}
