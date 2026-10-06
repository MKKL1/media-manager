package main

import "fmt"

func describe() any {
	return map[string]any{
		"formats": []string{"default:tv"},
		"fields":  []string{"title", "first_air_date", "season_number"},
	}
}

type entry struct {
	ID      string            `json:"id"`
	Values  map[string]string `json:"values"`
	Entries []entry           `json:"entries"`
}

func format(show entry) map[string]string {
	names := map[string]string{show.ID: withYear(show.Values["title"], show.Values["first_air_date"])}
	for i, group := range show.Entries {
		season := i + 1
		if _, err := fmt.Sscan(group.Values["season_number"], &season); err == nil {
			names[group.ID] = fmt.Sprintf("Season %02d", season)
			if season == 0 {
				names[group.ID] = "Specials"
			}
		} else {
			names[group.ID] = group.Values["title"]
		}
		for j, episode := range group.Entries {
			names[episode.ID] = fmt.Sprintf("S%02dE%02d", season, j+1)
			if title := episode.Values["title"]; title != "" {
				names[episode.ID] += " - " + title
			}
		}
	}
	return names
}

func withYear(title, date string) string {
	if len(date) >= 4 {
		return fmt.Sprintf("%s (%s)", title, date[:4])
	}
	return title
}
