package main

import "fmt"

func describe() any {
	return map[string]any{
		"formats": []string{"default:movie"},
		"fields":  []string{"title", "release_date"},
	}
}

type entry struct {
	ID      string            `json:"id"`
	Values  map[string]string `json:"values"`
	Entries []entry           `json:"entries"`
}

func format(movie entry) map[string]string {
	names := map[string]string{}
	var name func(entry)
	name = func(e entry) {
		names[e.ID] = e.Values["title"]
		if date := e.Values["release_date"]; len(date) >= 4 {
			names[e.ID] = fmt.Sprintf("%s (%s)", e.Values["title"], date[:4])
		}
		for _, child := range e.Entries {
			name(child)
		}
	}
	name(movie)
	return names
}
