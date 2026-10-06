package library_test

import (
	"context"
	"testing"
	"time"

	"server/internal/event"
	"server/internal/library"
)

func TestEveryChangeIsOneEvent(t *testing.T) {
	bus := &event.Bus{}
	got, _, err := bus.Subscribe("", func(event.Event) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	ctx := event.WithBy(context.Background(), "alice")
	lib, err := library.Open(ctx, nopStore{}, bus)
	if err != nil {
		t.Fatal(err)
	}
	show := library.ID{Provider: "tmdb", ProviderID: "tv:1"}
	ep := library.ID{Provider: "tmdb", ProviderID: "tv:1:e1"}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(lib.Put(ctx, library.Fetched{Provider: "tmdb", Arrangement: "aired", Top: show.ProviderID, CanHoldEntries: true, Entries: []library.Tree{{ProviderID: ep.ProviderID}}}))
	must(lib.Put(ctx, library.Value{Entry: ep, Field: "title", Text: "Pilot"}))
	must(lib.Put(ctx, library.Value{Entry: ep, Field: "title", Text: "Pilot"}))
	_, err = lib.Remove(ctx, library.Value{Entry: ep, Field: "title"})
	must(err)
	if err := lib.Put(ctx, library.Link{From: show, To: ep, Kind: library.Same}); err == nil {
		t.Fatal("linking a group should be refused")
	}
	must(lib.Put(ctx, library.Choice{Entry: show, Chosen: true}))

	for _, want := range []string{
		"fetched.updated entries/tmdb:tv:1",
		"edit.updated entries/tmdb:tv:1:e1",
		"edit.deleted entries/tmdb:tv:1:e1",
		"edit.updated entries/tmdb:tv:1",
	} {
		select {
		case e := <-got:
			if e.Type+" "+e.Subject != want || e.By != "alice" {
				t.Fatalf("got %s %s by %s, want %s by alice", e.Type, e.Subject, e.By, want)
			}
		case <-time.After(time.Second):
			t.Fatalf("no event, want %s", want)
		}
	}
	select {
	case e := <-got:
		t.Fatalf("one event too many: %s %s", e.Type, e.Subject)
	case <-time.After(50 * time.Millisecond):
	}
}
