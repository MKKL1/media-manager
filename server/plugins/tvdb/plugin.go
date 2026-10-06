package tvdb

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"server/internal/library"
	"server/internal/plugin"
)

const Provider library.Provider = "tvdb"

const (
	Aired     = "aired"
	DVD       = "dvd"
	Absolute  = "absolute"
	Alternate = "alternate"
	Regional  = "regional"
	AltDVD    = "altdvd"
)

var seasonTypeSlugs = map[string]string{
	Aired: "official", DVD: "dvd", Absolute: "absolute", Alternate: "alternate", Regional: "regional", AltDVD: "altdvd",
}

const language = "eng"

type Plugin struct{ client *Client }

func NewPlugin(apiKey, pin string) *Plugin { return &Plugin{NewClient(apiKey, pin)} }

var (
	_ plugin.Plugin       = (*Plugin)(nil)
	_ plugin.Arrangements = (*Plugin)(nil)
)

func (*Plugin) Provider() library.Provider { return Provider }

func (*Plugin) Arrangements() []string {
	return []string{Aired, DVD, Absolute, Alternate, Regional, AltDVD}
}

func (p *Plugin) Fetch(ctx context.Context, arrangement, ref string) (library.Tree, error) {
	kind, rest, _ := strings.Cut(ref, ":")
	number, _, _ := strings.Cut(rest, ":")
	tvdbID, err := strconv.Atoi(number)
	if err != nil || (kind != "movie" && kind != "series") {
		return library.Tree{}, fmt.Errorf("tvdb ref %q is not movie:<id> or series:<id>: %w", ref, plugin.ErrNotFound)
	}
	var top library.Tree
	switch {
	case kind == "movie" && arrangement == Aired:
		top, err = p.movie(ctx, tvdbID)
	case kind == "movie":
		return library.Tree{}, fmt.Errorf("a movie has no %s order: %w", arrangement, plugin.ErrNoSuchArrangement)
	default:
		top, err = p.series(ctx, tvdbID, arrangement)
	}
	if errors.Is(err, ErrNotFound) {
		err = fmt.Errorf("%w: %w", plugin.ErrNotFound, err)
	}
	return top, err
}

func (p *Plugin) movie(ctx context.Context, tvdbID int) (library.Tree, error) {
	m, err := p.client.GetMovie(ctx, tvdbID)
	if err != nil {
		return library.Tree{}, err
	}
	title, summary := m.Name, m.Overview
	if t, err := p.client.GetMovieTranslation(ctx, tvdbID, language); err == nil {
		title, summary = first(t.Name, title), first(t.Overview, summary)
	} else if !errors.Is(err, ErrNotFound) {
		return library.Tree{}, err
	}
	release := first(m.FirstRelease.Date, m.Year)
	v := fieldValues(
		"title", title,
		"summary", summary,
		"original_language", m.OriginalLanguage,
		"release_date", release,
		"genres", genreNames(m.Genres),
		"id.tvdb", strconv.Itoa(tvdbID),
	)
	addRemoteIDs(v, m.RemoteIDs)
	return library.Tree{ProviderID: fmt.Sprintf("movie:%d", tvdbID), Values: v}, nil
}

func (p *Plugin) series(ctx context.Context, tvdbID int, arrangement string) (library.Tree, error) {
	slug, ok := seasonTypeSlugs[arrangement]
	if !ok {
		return library.Tree{}, fmt.Errorf("tvdb has no %q order: %w", arrangement, plugin.ErrNoSuchArrangement)
	}
	s, err := p.client.GetSeries(ctx, tvdbID)
	if err != nil {
		return library.Tree{}, err
	}
	episodes, err := p.client.GetEpisodes(ctx, tvdbID, slug, language)
	if err != nil {
		return library.Tree{}, err
	}
	if len(episodes) == 0 {
		return library.Tree{}, fmt.Errorf("series:%d has no %s order: %w", tvdbID, arrangement, plugin.ErrNoSuchArrangement)
	}

	title, summary := s.Name, s.Overview
	if t, err := p.client.GetSeriesTranslation(ctx, tvdbID, language); err == nil {
		title, summary = first(t.Name, title), first(t.Overview, summary)
	} else if !errors.Is(err, ErrNotFound) {
		return library.Tree{}, err
	}
	v := fieldValues(
		"title", title,
		"summary", summary,
		"original_language", s.OriginalLanguage,
		"first_air_date", s.FirstAired,
		"status", s.Status.Name,
		"genres", genreNames(s.Genres),
		"id.tvdb", strconv.Itoa(tvdbID),
	)
	if s.Name != title {
		v["original_title"] = s.Name
	}
	addRemoteIDs(v, s.RemoteIDs)

	topID := fmt.Sprintf("series:%d", tvdbID)
	if arrangement != Aired {
		topID = fmt.Sprintf("series:%d:%s", tvdbID, arrangement)
	}
	top := library.Tree{ProviderID: topID, CanHoldEntries: true, Values: v}

	seen := map[int]bool{}
	var numbers []int
	bySeason := map[int][]Episode{}
	for _, ep := range episodes {
		if seen[ep.ID] {
			continue
		}
		seen[ep.ID] = true
		if _, ok := bySeason[ep.SeasonNumber]; !ok {
			numbers = append(numbers, ep.SeasonNumber)
		}
		bySeason[ep.SeasonNumber] = append(bySeason[ep.SeasonNumber], ep)
	}
	slices.Sort(numbers)
	for _, n := range numbers {
		eps := bySeason[n]
		slices.SortStableFunc(eps, func(a, b Episode) int { return a.Number - b.Number })
		name := fmt.Sprintf("Season %d", n)
		for _, season := range s.Seasons {
			if season.Type.Type == slug && season.Number == n && season.Name != "" {
				name = season.Name
			}
		}
		group := library.Tree{
			ProviderID:     fmt.Sprintf("%s:s%d", topID, n),
			CanHoldEntries: true,
			Values:         fieldValues("title", name, "season_number", strconv.Itoa(n)),
		}
		for _, ep := range eps {
			group.Entries = append(group.Entries, episode(ep))
		}
		top.Entries = append(top.Entries, group)
	}
	return top, nil
}

func episode(ep Episode) library.Tree {
	abs := ""
	if ep.AbsoluteNumber != 0 {
		abs = strconv.Itoa(ep.AbsoluteNumber)
	}
	return library.Tree{
		ProviderID: fmt.Sprintf("episode:%d", ep.ID),
		Values: fieldValues(
			"title", ep.Name,
			"summary", ep.Overview,
			"air_date", ep.Aired,
			"season_number", strconv.Itoa(ep.SeasonNumber),
			"episode_number", strconv.Itoa(ep.Number),
			"absolute_number", abs,
		),
	}
}

func addRemoteIDs(v map[library.Field]string, remote []RemoteID) {
	for _, r := range remote {
		switch r.SourceName {
		case "IMDB":
			v["id.imdb"] = r.ID
		case "TheMovieDB.com":
			v["id.tmdb"] = r.ID
		}
	}
}

func genreNames(genres []Genre) string {
	names := make([]string, 0, len(genres))
	for _, g := range genres {
		names = append(names, g.Name)
	}
	return strings.Join(names, ", ")
}

func first(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func fieldValues(kv ...string) map[library.Field]string {
	m := map[library.Field]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		if kv[i+1] != "" {
			m[library.Field(kv[i])] = kv[i+1]
		}
	}
	return m
}
