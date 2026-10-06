package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

var (
	errNotFound          = errors.New("not found")
	errNoSuchArrangement = errors.New("arrangement not offered")
)

const (
	Aired           = "aired"
	OriginalAirDate = "original-air-date"
	Absolute        = "absolute"
	DVD             = "dvd"
	Digital         = "digital"
	StoryArc        = "story-arc"
	Production      = "production"
	TV              = "tv"
)

var episodeGroupTypes = map[string]int{
	OriginalAirDate: 1, Absolute: 2, DVD: 3, Digital: 4, StoryArc: 5, Production: 6, TV: 7,
}

const (
	tagTV    = "default:tv"
	tagMovie = "default:movie"
	tagAnime = "default:anime"
)

type info struct {
	Provider     string   `json:"provider"`
	Arrangements []string `json:"arrangements"`
	Tags         []string `json:"tags"`
	CanSearch    bool     `json:"can_search"`
}

func describe() info {
	return info{
		Provider:     "tmdb",
		Arrangements: []string{Aired, OriginalAirDate, Absolute, DVD, Digital, StoryArc, Production, TV},
		Tags:         []string{tagTV, tagMovie, tagAnime},
		CanSearch:    true,
	}
}

type entry struct {
	ID             string            `json:"id"`
	CanHoldEntries bool              `json:"can_hold_entries,omitempty"`
	Values         map[string]string `json:"values,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Entries        []entry           `json:"entries,omitempty"`
}

type searchResult struct {
	Ref    string            `json:"ref"`
	Values map[string]string `json:"values,omitempty"`
	Tags   []string          `json:"tags,omitempty"`
}

type tmdb struct {
	get func(path string, query url.Values) ([]byte, error)
}

func (t tmdb) getJSON(v any, path string, query url.Values) error {
	body, err := t.get(path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func (t tmdb) fetch(arrangement, ref string) (entry, error) {
	kind, number, _ := strings.Cut(ref, ":")
	number, _, _ = strings.Cut(number, ":")
	tmdbID, err := strconv.Atoi(number)
	if err != nil {
		return entry{}, fmt.Errorf("%w: tmdb ref %q is not movie:<id> or tv:<id>", errNotFound, ref)
	}
	switch {
	case kind == "movie" && arrangement == Aired:
		return t.movie(tmdbID)
	case kind == "movie":
		return entry{}, fmt.Errorf("%w: a movie has no %s order", errNoSuchArrangement, arrangement)
	case kind == "tv" && arrangement == Aired:
		return t.showAired(tmdbID)
	case kind == "tv":
		return t.showInEpisodeGroup(tmdbID, arrangement)
	}
	return entry{}, fmt.Errorf("%w: tmdb ref %q is not movie:<id> or tv:<id>", errNotFound, ref)
}

func (t tmdb) movie(tmdbID int) (entry, error) {
	var m MovieDetails
	if err := t.getJSON(&m, "/3/movie/"+strconv.Itoa(tmdbID), url.Values{"append_to_response": {"external_ids"}}); err != nil {
		return entry{}, err
	}
	v := fieldValues(
		"title", m.Title,
		"summary", m.Overview,
		"original_title", m.OriginalTitle,
		"original_language", m.OriginalLanguage,
		"release_date", m.ReleaseDate,
		"genres", genreNames(m.Genres),
		"id.tmdb", strconv.Itoa(m.ID),
		"id.imdb", m.IMDbID,
	)
	return entry{ID: fmt.Sprintf("movie:%d", tmdbID), Values: v, Tags: tags(tagMovie, genreIDs(m.Genres), m.OriginalLanguage)}, nil
}

func tags(kind string, genres []int, language string) []string {
	if slices.Contains(genres, 16) && language == "ja" {
		return []string{kind, tagAnime}
	}
	return []string{kind}
}

func showValues(show *TVDetails) map[string]string {
	v := fieldValues(
		"title", show.Name,
		"summary", show.Overview,
		"original_title", show.OriginalName,
		"original_language", show.OriginalLanguage,
		"first_air_date", show.FirstAirDate,
		"status", show.Status,
		"genres", genreNames(show.Genres),
		"id.tmdb", strconv.Itoa(show.ID),
	)
	if x := show.ExternalIDs; x != nil {
		if x.IMDbID != "" {
			v["id.imdb"] = x.IMDbID
		}
		if x.TVDBID != 0 {
			v["id.tvdb"] = strconv.Itoa(x.TVDBID)
		}
	}
	return v
}

func (t tmdb) show(tmdbID int) (*TVDetails, error) {
	var show TVDetails
	err := t.getJSON(&show, "/3/tv/"+strconv.Itoa(tmdbID), url.Values{"append_to_response": {"external_ids,episode_groups"}})
	return &show, err
}

func showTags(show *TVDetails) []string {
	return tags(tagTV, genreIDs(show.Genres), show.OriginalLanguage)
}

func (t tmdb) showAired(tmdbID int) (entry, error) {
	show, err := t.show(tmdbID)
	if err != nil {
		return entry{}, err
	}
	top := entry{ID: fmt.Sprintf("tv:%d", tmdbID), CanHoldEntries: true, Values: showValues(show), Tags: showTags(show)}
	for _, s := range show.Seasons {
		var detail Season
		if err := t.getJSON(&detail, fmt.Sprintf("/3/tv/%d/season/%d", tmdbID, s.SeasonNumber), nil); err != nil {
			return entry{}, fmt.Errorf("season %d: %w", s.SeasonNumber, err)
		}
		name := detail.Name
		if name == "" {
			name = fmt.Sprintf("Season %d", s.SeasonNumber)
		}
		season := entry{
			ID:             fmt.Sprintf("tv:%d:s%d", tmdbID, s.SeasonNumber),
			CanHoldEntries: true,
			Values:         fieldValues("title", name, "summary", detail.Overview, "season_number", strconv.Itoa(s.SeasonNumber)),
		}
		for _, ep := range detail.Episodes {
			season.Entries = append(season.Entries, episode(tmdbID, ep))
		}
		top.Entries = append(top.Entries, season)
	}
	return top, nil
}

func (t tmdb) showInEpisodeGroup(tmdbID int, arrangement string) (entry, error) {
	groupType, ok := episodeGroupTypes[arrangement]
	if !ok {
		return entry{}, fmt.Errorf("%w: tmdb has no %q order", errNoSuchArrangement, arrangement)
	}
	show, err := t.show(tmdbID)
	if err != nil {
		return entry{}, err
	}
	groupID := ""
	if show.EpisodeGroups != nil {
		for _, g := range show.EpisodeGroups.Results {
			if g.Type == groupType {
				groupID = g.ID
				break
			}
		}
	}
	if groupID == "" {
		return entry{}, fmt.Errorf("%w: tv:%d has no %s order", errNoSuchArrangement, tmdbID, arrangement)
	}
	var detail EpisodeGroupDetailResponse
	if err := t.getJSON(&detail, "/3/tv/episode_group/"+groupID, nil); err != nil {
		return entry{}, fmt.Errorf("episode group %s: %w", groupID, err)
	}
	v := showValues(show)
	v["order_name"] = detail.Name
	top := entry{ID: fmt.Sprintf("tv:%d:%s", tmdbID, arrangement), CanHoldEntries: true, Values: v, Tags: showTags(show)}
	for _, g := range detail.Groups {
		group := entry{
			ID:             fmt.Sprintf("tv:%d:group:%s", tmdbID, g.ID),
			CanHoldEntries: true,
			Values:         fieldValues("title", g.Name, "order", strconv.Itoa(g.Order)),
		}
		for _, ep := range g.Episodes {
			group.Entries = append(group.Entries, episode(tmdbID, ep))
		}
		top.Entries = append(top.Entries, group)
	}
	return top, nil
}

func episode(tmdbID int, ep Episode) entry {
	return entry{
		ID: fmt.Sprintf("tv:%d:s%d:e%d", tmdbID, ep.SeasonNumber, ep.EpisodeNumber),
		Values: fieldValues(
			"title", ep.Name,
			"summary", ep.Overview,
			"air_date", ep.AirDate,
			"season_number", strconv.Itoa(ep.SeasonNumber),
			"episode_number", strconv.Itoa(ep.EpisodeNumber),
		),
	}
}

func fieldValues(kv ...string) map[string]string {
	m := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			m[kv[i]] = kv[i+1]
		}
	}
	return m
}

func genreNames(genres []Genre) string {
	names := make([]string, len(genres))
	for i, g := range genres {
		names[i] = g.Name
	}
	return strings.Join(names, ", ")
}

func genreIDs(genres []Genre) []int {
	ids := make([]int, len(genres))
	for i, g := range genres {
		ids[i] = g.ID
	}
	return ids
}

func (t tmdb) search(query string) ([]searchResult, error) {
	var found struct {
		Results []MultiSearchResult `json:"results"`
	}
	if err := t.getJSON(&found, "/3/search/multi", url.Values{"query": {query}, "page": {"1"}}); err != nil {
		return nil, err
	}
	results := []searchResult{}
	for _, f := range found.Results {
		switch f.MediaType {
		case "tv":
			results = append(results, searchResult{Ref: fmt.Sprintf("tv:%d", f.ID), Tags: tags(tagTV, f.GenreIDs, f.OriginalLanguage), Values: fieldValues(
				"title", f.Name, "original_title", f.OriginalName, "summary", f.Overview, "first_air_date", f.FirstAirDate,
			)})
		case "movie":
			results = append(results, searchResult{Ref: fmt.Sprintf("movie:%d", f.ID), Tags: tags(tagMovie, f.GenreIDs, f.OriginalLanguage), Values: fieldValues(
				"title", f.Title, "original_title", f.OriginalTitle, "summary", f.Overview, "release_date", f.ReleaseDate,
			)})
		}
	}
	return results, nil
}
