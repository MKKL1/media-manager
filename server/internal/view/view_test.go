package view_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"testing"

	"server/internal/event"
	"server/internal/library"
	"server/internal/plugin"
	"server/internal/view"
)

type asTV struct{}

func (asTV) Formats() []plugin.Tag       { return []plugin.Tag{"default:tv"} }
func (asTV) FieldsRead() []library.Field { return []library.Field{"title"} }
func (asTV) Names(_ context.Context, _ plugin.Tag, work view.Entry) (map[library.ID]string, error) {
	names := map[library.ID]string{work.ID: work.Values["title"]}
	var walk func(view.Entry)
	walk = func(e view.Entry) {
		for i, child := range e.Entries {
			names[child.ID] = fmt.Sprint(i+1, " ", child.Values["title"])
			walk(child)
		}
	}
	walk(work)
	return names, nil
}

func id(s string) library.ID { return library.ID{Provider: "tmdb", ProviderID: s} }

func episode(n, title string) library.Tree {
	return library.Tree{ProviderID: "tv:1:s1:e" + n, Values: map[library.Field]string{"title": title}}
}

func TestShow(t *testing.T) {
	ctx := context.Background()
	lib, err := library.Open(ctx, nopStore{}, &event.Bus{})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(lib.Put(ctx, library.Fetched{
		Provider: "tmdb", Arrangement: "aired", Top: "tv:1", CanHoldEntries: true, Values: map[library.Field]string{"title": "Show", "tags": "default:tv"},
		Entries: []library.Tree{{ProviderID: "tv:1:s1", CanHoldEntries: true, Values: map[library.Field]string{"title": "Season"},
			Entries: []library.Tree{episode("1", "Pilot"), episode("2", "Second")}}},
	}))
	must(lib.Put(ctx, library.Fetched{
		Provider: "tmdb", Arrangement: "dvd", Top: "tv:1:dvd", CanHoldEntries: true, Values: map[library.Field]string{"title": "Show", "tags": "default:tv"},
		Entries: []library.Tree{{ProviderID: "tv:1:d1", CanHoldEntries: true, Values: map[library.Field]string{"title": "Disc"},
			Entries: []library.Tree{episode("2", "Second"), episode("1", "Pilot (DVD)")}}},
	}))
	must(lib.Put(ctx, library.Choice{Entry: id("tv:1"), Chosen: true}))
	views := &view.Views{Library: lib, Types: func() []view.Type { return []view.Type{asTV{}} }}

	names := func(s view.Shown) (all []string) {
		var walk func(view.Entry)
		walk = func(e view.Entry) {
			all = append(all, e.Name)
			for _, c := range e.Entries {
				walk(c)
			}
		}
		walk(s.Work)
		return all
	}
	aired, err := views.Show(ctx, id("tv:1"), "", "")
	must(err)
	if got := fmt.Sprint(aired.Arrangement, aired.Type, names(aired)); got != "tmdb/aireddefault:tv[Show 1 Season 1 Pilot 2 Second]" {
		t.Errorf("aired: %s", got)
	}
	dvd, err := views.Show(ctx, id("tv:1"), "dvd", "")
	must(err)
	if got := fmt.Sprint(dvd.Arrangement, dvd.Work.ID.String(), names(dvd)); got != "tmdb/dvdtmdb:tv:1:dvd[Show 1 Disc 1 Second 2 Pilot (DVD)]" {
		t.Errorf("dvd: %s", got)
	}

	must(lib.Put(ctx, library.Value{Entry: id("tv:1:s1:e1"), Field: "title", Text: "Serenity"}))
	dvd, err = views.Show(ctx, id("tv:1"), "dvd", view.FirstTypedTag)
	must(err)
	if got := names(dvd)[3]; got != "2 Serenity" {
		t.Errorf("an edit wins in every view: %s", got)
	}

	for _, c := range []struct {
		id          string
		arrangement view.Arrangement
		as          plugin.Tag
		kind        view.NotFoundKind
	}{
		{"tv:2", "", "", "work"},
		{"tv:1:s1", "", "", "work"},
		{"tv:1:dvd", "", "", "work"},
		{"tv:1", "tvdb/default", "", "arrangement"},
		{"tv:1", "", "default:anime", "type"},
	} {
		_, err := views.Show(ctx, id(c.id), c.arrangement, c.as)
		var nf view.NotFound
		if !errors.Is(err, view.ErrNotFound) || !errors.As(err, &nf) || nf.Kind != c.kind {
			t.Errorf("%+v: %v", c, err)
		}
	}
}

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}
