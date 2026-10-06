package anime_list

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/sony/gobreaker"

	"server/internal/library"
	"server/internal/plugin"
)

const Provider library.Provider = "anime-lists"

const listURL = "https://raw.githubusercontent.com/Anime-Lists/anime-lists/master/anime-list.xml"

const listKeptFor = 24 * time.Hour

type Plugin struct {
	client  *http.Client
	breaker *gobreaker.CircuitBreaker
	url     string

	lock         sync.Mutex
	byAniDBID    map[int]*xmlAnime
	downloadedAt time.Time
}

var _ plugin.Mappings = (*Plugin)(nil)

func NewPlugin(url string) *Plugin {
	if url == "" {
		url = listURL
	}
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	retryClient.HTTPClient = &http.Client{
		Timeout: 30 * time.Second,
	}
	return &Plugin{
		client: retryClient.StandardClient(),
		breaker: gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name: "anime-list",
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.Requests >= 5 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5
			},
			Timeout: 10 * time.Second,
		}),
		url: url,
	}
}

func (*Plugin) Provider() library.Provider { return Provider }

func (p *Plugin) Refs(ctx context.Context, lib *library.Library) ([]string, error) {
	list, err := p.list(ctx)
	if err != nil {
		return nil, err
	}
	tvdbShows, tmdbShows := showIDs(lib, "tvdb", "id.tvdb"), showIDs(lib, "tmdb", "id.tmdb")
	var refs []int
	for id, a := range list {
		if tvdbShows[a.TVDBID] && tmdbShows[a.TMDBTv] {
			refs = append(refs, id)
		}
	}
	slices.Sort(refs)
	out := make([]string, len(refs))
	for i, id := range refs {
		out[i] = "anidb:" + strconv.Itoa(id)
	}
	return out, nil
}

func (p *Plugin) FetchMapping(ctx context.Context, lib *library.Library, ref string) (library.Mapping, error) {
	number, ok := strings.CutPrefix(ref, "anidb:")
	anidbID, err := strconv.Atoi(number)
	if !ok || err != nil {
		return library.Mapping{}, fmt.Errorf("anime-lists ref %q is not anidb:<id>: %w", ref, plugin.ErrNotFound)
	}
	list, err := p.list(ctx)
	if err != nil {
		return library.Mapping{}, err
	}
	a, ok := list[anidbID]
	if !ok {
		return library.Mapping{}, fmt.Errorf("anidb %d is not listed: %w", anidbID, plugin.ErrNotFound)
	}
	links := library.Mapping{Provider: Provider, OwnID: ref}
	tvdbShow, foundTVDB := findShow(lib, "tvdb", "id.tvdb", a.TVDBID)
	tmdbShow, foundTMDB := findShow(lib, "tmdb", "id.tmdb", a.TMDBTv)
	if !foundTVDB || !foundTMDB {
		return links, nil
	}
	tvdb, tmdb := placesOf(a, "tvdb"), placesOf(a, "tmdb")
	why := []library.Evidence{library.Evidence("anime-lists anidb " + strconv.Itoa(anidbID))}
	linked := map[[2]library.ID]bool{}
	link := func(from, to library.ID, kind library.LinkKind) {
		key := [2]library.ID{from, to}
		if idLess(to, from) {
			key = [2]library.ID{to, from}
		}
		if !linked[key] {
			linked[key] = true
			links.Links = append(links.Links, library.Link{From: from, To: to, Kind: kind, Why: why})
		}
	}
	for _, e := range aniDBEpisodes(tvdb, tmdb, tvdbShow, tmdbShow, episodeCountAtMost(list, a)) {
		tvdbEpisodes, tmdbEpisodes := tvdbShow.find(tvdb.where(e)), tmdbShow.find(tmdb.where(e))
		switch {
		case len(tvdbEpisodes) == 1 && len(tmdbEpisodes) == 1:
			link(tvdbEpisodes[0], tmdbEpisodes[0], library.Same)
		case len(tvdbEpisodes) == 1 && len(tmdbEpisodes) > 1:
			for _, part := range tmdbEpisodes {
				link(tvdbEpisodes[0], part, library.Contains)
			}
		case len(tmdbEpisodes) == 1 && len(tvdbEpisodes) > 1:
			for _, part := range tvdbEpisodes {
				link(tmdbEpisodes[0], part, library.Contains)
			}

		}
	}
	return links, nil
}

func idLess(a, b library.ID) bool {
	return a.Provider < b.Provider || (a.Provider == b.Provider && a.ProviderID < b.ProviderID)
}

func (p *Plugin) list(ctx context.Context) (map[int]*xmlAnime, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.byAniDBID != nil && time.Since(p.downloadedAt) < listKeptFor {
		return p.byAniDBID, nil
	}
	parsed, err := p.breaker.Execute(func() (any, error) { return p.download(ctx) })
	if err != nil {
		return nil, fmt.Errorf("anime-lists: %w", err)
	}
	p.byAniDBID, p.downloadedAt = parsed.(map[int]*xmlAnime), time.Now()
	return p.byAniDBID, nil
}

func (p *Plugin) download(ctx context.Context) (map[int]*xmlAnime, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var list xmlAnimeList
	if err := xml.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	byID := make(map[int]*xmlAnime, len(list.Anime))
	for i := range list.Anime {
		byID[list.Anime[i].AniDBID] = &list.Anime[i]
	}
	return byID, nil
}

type aniDBEpisode struct{ season, number int }

type place struct {
	absolute bool
	season   int
	numbers  []int
}

type places struct {
	defaultSeason string
	offset        int
	listed        map[aniDBEpisode]place
	ranges        []xmlMapping
	season        func(xmlMapping) string
}

func placesOf(a *xmlAnime, provider string) places {
	ps := places{listed: map[aniDBEpisode]place{}}
	if provider == "tvdb" {
		ps.defaultSeason, ps.offset = a.DefaultTVDBSeason, a.EpisodeOffset
		ps.season = func(m xmlMapping) string { return m.TVDBSeason }
	} else {
		ps.defaultSeason, ps.offset = a.TMDBSeason, a.TMDBOffset
		ps.season = func(m xmlMapping) string { return m.TMDBSeason }
	}
	if a.MappingList == nil {
		return ps
	}
	for _, m := range a.MappingList.Mappings {
		season := ps.season(m)
		if season == "" {
			continue
		}
		if strings.TrimSpace(m.Content) == "" {
			ps.ranges = append(ps.ranges, m)
			continue
		}
		for _, pairText := range strings.Split(m.Content, ";") {
			from, to, ok := strings.Cut(pairText, "-")
			if !ok {
				continue
			}
			n, err := strconv.Atoi(from)
			if err != nil {
				continue
			}
			pl := seasonPlace(season)
			for _, t := range strings.Split(to, "+") {
				if t, err := strconv.Atoi(t); err == nil && t > 0 {
					pl.numbers = append(pl.numbers, t)
				}
			}
			ps.listed[aniDBEpisode{m.AniDBSeason, n}] = pl
		}
	}
	return ps
}

func seasonPlace(season string) place {
	if season == "a" {
		return place{absolute: true}
	}
	n, _ := strconv.Atoi(season)
	return place{season: n}
}

func (ps places) where(e aniDBEpisode) place {
	if pl, ok := ps.listed[e]; ok {
		return pl
	}
	for _, m := range ps.ranges {
		if m.AniDBSeason == e.season && e.number >= m.Start && (m.End == 0 || e.number <= m.End) {
			pl := seasonPlace(ps.season(m))
			pl.numbers = []int{e.number + m.Offset}
			return pl
		}
	}
	if e.season != 1 || ps.defaultSeason == "" {
		return place{}
	}
	pl := seasonPlace(ps.defaultSeason)
	pl.numbers = []int{e.number + ps.offset}
	return pl
}

func episodeCountAtMost(list map[int]*xmlAnime, a *xmlAnime) int {
	count := 10_000
	for _, b := range list {
		if b.TVDBID == a.TVDBID && b.DefaultTVDBSeason == a.DefaultTVDBSeason && a.DefaultTVDBSeason != "" && b.EpisodeOffset > a.EpisodeOffset {
			count = min(count, b.EpisodeOffset-a.EpisodeOffset)
		}
		if b.TMDBTv == a.TMDBTv && b.TMDBSeason == a.TMDBSeason && a.TMDBSeason != "" && b.TMDBOffset > a.TMDBOffset {
			count = min(count, b.TMDBOffset-a.TMDBOffset)
		}
	}
	return count
}

func aniDBEpisodes(tvdb, tmdb places, tvdbShow, tmdbShow show, countAtMost int) []aniDBEpisode {
	var out []aniDBEpisode
	lastListed := 0
	for _, ps := range []places{tvdb, tmdb} {
		for e := range ps.listed {
			if e.season == 1 {
				lastListed = max(lastListed, e.number)
			} else {
				out = append(out, e)
			}
		}
		for _, m := range ps.ranges {
			if m.AniDBSeason == 1 {
				lastListed = max(lastListed, m.Start, m.End)
			}
		}
	}
	for n := 1; n <= countAtMost; n++ {
		e := aniDBEpisode{1, n}
		if n > lastListed && len(tvdbShow.find(tvdb.where(e))) == 0 && len(tmdbShow.find(tmdb.where(e))) == 0 {
			break
		}
		out = append(out, e)
	}
	slices.SortFunc(out, func(x, y aniDBEpisode) int {
		if x.season != y.season {
			return x.season - y.season
		}
		return x.number - y.number
	})
	return slices.Compact(out)
}

type show struct {
	bySeasonAndNumber map[[2]int]library.ID
	byAbsoluteNumber  map[int]library.ID
}

func (s show) find(pl place) []library.ID {
	var ids []library.ID
	for _, n := range pl.numbers {
		var id library.ID
		var ok bool
		if pl.absolute {
			id, ok = s.byAbsoluteNumber[n]
		} else {
			id, ok = s.bySeasonAndNumber[[2]int{pl.season, n}]
		}
		if !ok {
			return nil
		}
		ids = append(ids, id)
	}
	return ids
}

func showIDs(lib *library.Library, provider library.Provider, idField library.Field) map[string]bool {
	ids := map[string]bool{}
	aired, ok := airedOf(lib, provider)
	if !ok {
		return ids
	}
	for top := range aired.Top() {
		if id, ok := airedValue(aired, top, idField); ok {
			ids[id] = true
		}
	}
	return ids
}

func findShow(lib *library.Library, provider library.Provider, idField library.Field, id string) (show, bool) {
	aired, ok := airedOf(lib, provider)
	if !ok {
		return show{}, false
	}
	for top := range aired.Top() {
		if v, ok := airedValue(aired, top, idField); !ok || v != id {
			continue
		}
		s := show{bySeasonAndNumber: map[[2]int]library.ID{}, byAbsoluteNumber: map[int]library.ID{}}
		g, ok := top.AsGroup()
		if !ok {
			return s, true
		}
		for season := range g.Entries() {
			seasonGroup, ok := season.AsGroup()
			if !ok {
				continue
			}
			for ep := range seasonGroup.Entries() {
				s.add(aired, ep)
			}
		}
		return s, true
	}
	return show{}, false
}

func (s show) add(aired library.Arrangement, ep library.Entry) {
	number := func(f library.Field) (int, bool) {
		v, ok := airedValue(aired, ep, f)
		if !ok {
			return 0, false
		}
		n, err := strconv.Atoi(v)
		return n, err == nil
	}
	if season, ok := number("season_number"); ok {
		if n, ok := number("episode_number"); ok {
			s.bySeasonAndNumber[[2]int{season, n}] = ep.ID()
		}
	}
	if n, ok := number("absolute_number"); ok {
		s.byAbsoluteNumber[n] = ep.ID()
	}
}

func airedOf(lib *library.Library, provider library.Provider) (library.Arrangement, bool) {
	for a := range lib.Arrangements(provider) {
		if a.Name() == "aired" {
			return a, true
		}
	}
	return library.Arrangement{}, false
}

func airedValue(aired library.Arrangement, e library.Entry, f library.Field) (string, bool) {
	for s, text := range e.Values(f) {
		if fetched, ok := s.(library.Fetched); ok && fetched.Provider == aired.Provider() && fetched.Arrangement == aired.Name() {
			return text, true
		}
	}
	return "", false
}
