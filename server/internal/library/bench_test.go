package library_test

import (
	"context"
	"fmt"
	"iter"
	"runtime"
	"testing"

	"server/internal/event"
	"server/internal/library"
)

func benchLibrary(b *testing.B, shows int) *library.Library {
	b.Helper()
	l, err := library.Open(ctx, nopStore{}, &event.Bus{})
	must(b, err)
	providers := []library.Provider{tmdb, tvmaze, "anilist"}
	for _, p := range providers {
		for s := range shows {
			show := library.Fetched{Provider: p, Arrangement: "aired", Top: fmt.Sprint("show", s), CanHoldEntries: true}
			for season := range 5 {
				g := library.Tree{ProviderID: fmt.Sprint("s", s, "-", season), CanHoldEntries: true}
				for e := range 20 {
					g.Entries = append(g.Entries, library.Tree{ProviderID: fmt.Sprint("e", s, "-", season, "-", e), Values: map[library.Field]string{
						title: fmt.Sprint("Episode ", e, " of season ", season, " of show ", s), "air date": "2024-01-01",
					}})
				}
				show.Entries = append(show.Entries, g)
			}
			put(b, l, show)
		}
	}
	for s := range shows {
		for season := range 5 {
			for e := range 20 {
				ep := fmt.Sprint("e", s, "-", season, "-", e)
				why := []library.Evidence{"air date", "title"}
				put(b, l,
					library.Link{From: id(tmdb, ep), To: id(tvmaze, ep), Kind: library.Same, Why: why},
					library.Link{From: id(tvmaze, ep), To: id("anilist", ep), Kind: library.Same, Why: why})
			}
		}
	}
	for s := range shows {
		put(b, l, library.Choice{Entry: id(tmdb, fmt.Sprint("show", s)), Chosen: true})
	}
	return l
}

const benchShows = 500

func BenchmarkNewSnapshot(b *testing.B) {
	l := benchLibrary(b, benchShows)
	b.ResetTimer()
	i := 0
	for b.Loop() {
		i++
		put(b, l, library.Value{Entry: id(tmdb, "e0-0-0"), Field: title, Text: fmt.Sprint(i)})
		for range l.Works() {
			break
		}
	}
}

func BenchmarkPutOneShow(b *testing.B) {
	l := benchLibrary(b, benchShows)
	show := library.Fetched{Provider: tmdb, Arrangement: "aired", Top: "show0", CanHoldEntries: true}
	for season := range 5 {
		g := library.Tree{ProviderID: fmt.Sprint("s0-", season), CanHoldEntries: true}
		for e := range 20 {
			g.Entries = append(g.Entries, library.Tree{ProviderID: fmt.Sprint("e0-", season, "-", e), Values: map[library.Field]string{title: "x"}})
		}
		show.Entries = append(show.Entries, g)
	}
	b.ResetTimer()
	for b.Loop() {
		put(b, l, show)
	}
}

func BenchmarkItemValues(b *testing.B) {
	l := benchLibrary(b, benchShows)
	for range l.Works() {
		break
	}
	b.ResetTimer()
	for b.Loop() {
		i, _ := entry(b, l, id(tvmaze, "e7-3-9")).AsItem()
		for e := range i.Entries() {
			for _, text := range e.Values(title) {
				_ = text
			}
		}
	}
}

func BenchmarkWalkWorks(b *testing.B) {
	l := benchLibrary(b, benchShows)
	var walk func(g library.Group) int
	walk = func(g library.Group) (n int) {
		for e := range g.Entries() {
			n++
			if sub, ok := e.AsGroup(); ok {
				n += walk(sub)
			} else {
				e.AsItem()
			}
		}
		return n
	}
	b.ResetTimer()
	for b.Loop() {
		for top := range l.Works() {
			g, _ := top.AsGroup()
			walk(g)
		}
	}
}

func BenchmarkMemory(b *testing.B) {
	for b.Loop() {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		l := benchLibrary(b, benchShows)
		for range l.Works() {
			break
		}
		runtime.GC()
		runtime.ReadMemStats(&after)
		b.ReportMetric(float64(after.HeapAlloc-before.HeapAlloc)/(3*benchShows*106), "B/entry")
		runtime.KeepAlive(l)
	}
}

type nopStore struct{}

func (nopStore) Put(context.Context, library.Statement) error    { return nil }
func (nopStore) Remove(context.Context, library.Statement) error { return nil }
func (nopStore) All(context.Context) iter.Seq2[library.Statement, error] {
	return func(func(library.Statement, error) bool) {}
}
