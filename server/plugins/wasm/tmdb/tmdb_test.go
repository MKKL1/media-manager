package main

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"testing"
)

var fakeTMDB = tmdb{get: func(path string, _ url.Values) ([]byte, error) {
	body, ok := map[string]string{
		"/3/tv/1": `{"id":1,"name":"Show","external_ids":{"imdb_id":"tt1"},
			"seasons":[{"season_number":1}],
			"episode_groups":{"results":[{"id":"g","type":3}]}}`,
		"/3/tv/1/season/1": `{"season_number":1,"episodes":[
			{"season_number":1,"episode_number":1,"name":"Pilot"},
			{"season_number":1,"episode_number":2,"name":"Second"}]}`,
		"/3/tv/episode_group/g": `{"id":"g","name":"DVD","groups":[{"id":"d1","name":"Disc 1","order":1,"episodes":[
			{"season_number":1,"episode_number":2,"name":"Second"},
			{"season_number":1,"episode_number":1,"name":"Pilot"}]}]}`,
		"/3/tv/2": `{"id":2,"name":"Other","original_language":"ja","genres":[{"id":16,"name":"Animation"}],"seasons":[]}`,
	}[path]
	if !ok {
		return nil, fmt.Errorf("%w: %s", errNotFound, path)
	}
	return []byte(body), nil
}}

func TestFetch(t *testing.T) {
	aired, err := fakeTMDB.fetch(Aired, "tv:1")
	if err != nil {
		t.Fatal(err)
	}
	if got := aired.Entries[0].Entries[0]; got.ID != "tv:1:s1:e1" || got.Values["title"] != "Pilot" {
		t.Errorf("first aired episode = %+v", got)
	}
	if !slices.Equal(aired.Tags, []string{tagTV}) {
		t.Errorf("tags = %v", aired.Tags)
	}
	dvd, err := fakeTMDB.fetch(DVD, "tv:1:dvd")
	if err != nil {
		t.Fatal(err)
	}
	if dvd.ID != "tv:1:dvd" || dvd.Entries[0].ID != "tv:1:group:d1" || dvd.Entries[0].Entries[0].ID != "tv:1:s1:e2" {
		t.Errorf("DVD order = %+v", dvd)
	}

	anime, err := fakeTMDB.fetch(Aired, "tv:2")
	if err != nil || !slices.Equal(anime.Tags, []string{tagTV, tagAnime}) {
		t.Errorf("Japanese animation: %v %v", anime.Tags, err)
	}
	if _, err := fakeTMDB.fetch(DVD, "tv:2"); !errors.Is(err, errNoSuchArrangement) {
		t.Errorf("show without a DVD order: %v", err)
	}
	if _, err := fakeTMDB.fetch(DVD, "movie:1"); !errors.Is(err, errNoSuchArrangement) {
		t.Errorf("movie in DVD order: %v", err)
	}
	for _, ref := range []string{"tv:3", "person:1"} {
		if _, err := fakeTMDB.fetch(Aired, ref); !errors.Is(err, errNotFound) {
			t.Errorf("%s: %v", ref, err)
		}
	}
}
