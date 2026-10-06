package event_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"server/internal/event"
)

func all(event.Event) bool { return true }

func types(events <-chan event.Event, n int) (got []string) {
	for range n {
		got = append(got, (<-events).Type)
	}
	return got
}

func TestBusDeliversInOrderAndResumes(t *testing.T) {
	var bus event.Bus
	ctx := event.WithBy(context.Background(), "alice")
	events, cancel, err := bus.Subscribe("", all)
	if err != nil {
		t.Fatal(err)
	}
	edits, _, _ := bus.Subscribe("", func(e event.Event) bool { return e.Type == "edit.updated" })
	bus.Publish(ctx, event.Event{Type: "fetched.updated"})
	bus.Publish(ctx, event.Event{Type: "edit.updated"})

	first := <-events
	if first.By != "alice" || first.ID == "" || first.Time.IsZero() {
		t.Fatalf("got %+v, want by alice with id and time", first)
	}
	if got := fmt.Sprint(first.Type, types(events, 1), types(edits, 1)); got != "fetched.updated[edit.updated] [edit.updated]" {
		t.Fatalf("got %s", got)
	}

	cancel()
	if _, open := <-events; open {
		t.Fatal("cancel should close the channel")
	}
	bus.Publish(ctx, event.Event{Type: "c"})
	resumed, _, err := bus.Subscribe(first.ID, all)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(types(resumed, 2)); got != "[edit.updated c]" {
		t.Fatalf("resumed %s", got)
	}
}

func TestBusCutsOffWhoFallsBehind(t *testing.T) {
	var bus event.Bus
	slow, _, _ := bus.Subscribe("", all)
	for range 1100 {
		bus.Publish(context.Background(), event.Event{Type: "x"})
	}
	n := 0
	for range slow {
		n++
	}
	if n == 0 || n > 100 {
		t.Fatalf("a slow subscriber got %d events before the cut-off", n)
	}
	for _, id := range []string{"elsewhere-1", "bad", ""} {
		_, _, err := bus.Subscribe(id, all)
		if want := id != ""; errors.Is(err, event.ErrTooOld) != want {
			t.Errorf("resume after %q: %v", id, err)
		}
	}
	ch, _, _ := bus.Subscribe("", all)
	bus.Publish(context.Background(), event.Event{})
	start, _, _ := strings.Cut((<-ch).ID, "-")
	if _, _, err := bus.Subscribe(start+"-1", all); !errors.Is(err, event.ErrTooOld) {
		t.Errorf("an event no longer kept should be too old: %v", err)
	}
}
