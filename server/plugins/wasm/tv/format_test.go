package main

import (
	"maps"
	"testing"
)

func TestFormat(t *testing.T) {
	ep := func(id, title string) entry { return entry{ID: id, Values: map[string]string{"title": title}} }
	aired := entry{ID: "show", Values: map[string]string{"title": "Firefly", "first_air_date": "2002-09-20"}, Entries: []entry{
		{ID: "s0", Values: map[string]string{"season_number": "0"}, Entries: []entry{ep("e0", "Here's How It Was")}},
		{ID: "s1", Values: map[string]string{"season_number": "1"}, Entries: []entry{ep("e1", "Serenity"), ep("e2", "")}},
		{ID: "disc", Values: map[string]string{"title": "Disc 3"}, Entries: []entry{ep("e3", "Trash")}},
	}}
	want := map[string]string{
		"show": "Firefly (2002)",
		"s0":   "Specials", "e0": "S00E01 - Here's How It Was",
		"s1": "Season 01", "e1": "S01E01 - Serenity", "e2": "S01E02",
		"disc": "Disc 3", "e3": "S03E01 - Trash",
	}
	if got := format(aired); !maps.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}
