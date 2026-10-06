package main

import "testing"

func TestFormat(t *testing.T) {
	got := format(entry{ID: "m", Values: map[string]string{"title": "The Matrix", "release_date": "1999-03-30"}})
	if got["m"] != "The Matrix (1999)" {
		t.Errorf("got %v", got)
	}
}
