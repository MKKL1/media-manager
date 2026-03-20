package tmdb

import (
	"context"
	"fmt"
	"server/internal/core"
	"server/internal/media/movie"
	"server/internal/media/tv"
	"server/internal/metadata"
	"strconv"
	"time"
)

var (
	_ movie.Searcher = (*Provider)(nil)
	_ movie.Fetcher  = (*Provider)(nil)
	_ tv.Searcher    = (*Provider)(nil)
	_ tv.Fetcher     = (*Provider)(nil)
)

type Provider struct {
	client *Client
}

func NewProvider(apiKey string) *Provider {
	return &Provider{client: NewClient(apiKey)}
}

func (p *Provider) SearchMovie(ctx context.Context, query movie.SearchQuery) ([]movie.SearchResult, error) {
	movies, err := p.client.SearchMovies(ctx, SearchMovieParams{
		Query: query.Title,
		Year:  query.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("tmdb search movies: %w", err)
	}

	results := make([]movie.SearchResult, 0, len(movies))
	for _, m := range movies {
		results = append(results, movie.SearchResult{
			ExternalID: core.NewExternalId(metadata.ProviderTMDBMovie, strconv.Itoa(m.ID)),
			Title:      m.Title,
			Year:       yearFromDate(m.ReleaseDate),
			Overview:   m.Overview,
			Poster:     m.PosterPath,
			Popularity: m.Popularity,
		})
	}
	return results, nil
}

func (p *Provider) GetMovie(ctx context.Context, id string) (*movie.ProviderMovie, error) {
	tmdbID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tmdb movie id %q: %w", id, err)
	}

	d, err := p.client.GetMovie(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("tmdb get movie %d: %w", tmdbID, err)
	}

	genres := make([]string, len(d.Genres))
	for i, g := range d.Genres {
		genres[i] = g.Name
	}

	countries := make([]string, len(d.ProductionCompanies))
	for i, c := range d.ProductionCompanies {
		countries[i] = c.OriginCountry
	}

	extIDs := p.movieExternalIDs(ctx, tmdbID)

	releaseDate, _ := parseDate(d.ReleaseDate)

	return &movie.ProviderMovie{
		ExternalID:       core.NewExternalId(metadata.ProviderTMDBMovie, strconv.Itoa(d.ID)),
		ExternalIDs:      extIDs,
		Title:            d.Title,
		OriginalTitle:    d.OriginalTitle,
		OriginalLanguage: d.OriginalLanguage,
		Overview:         d.Overview,
		Tagline:          d.Tagline,
		Status:           d.Status,
		ReleaseDate:      releaseDate,
		Year:             yearFromDate(d.ReleaseDate),
		Runtime:          d.Runtime,
		Genres:           genres,
		OriginCountry:    countries,
		Poster:           d.PosterPath,
		Backdrop:         d.BackdropPath,
		Rating:           float32(d.VoteAverage),
		VoteCount:        d.VoteCount,
		Popularity:       d.Popularity,
		Budget:           d.Budget,
		Revenue:          d.Revenue,
	}, nil
}

func (p *Provider) SearchTV(ctx context.Context, query tv.SearchQuery) ([]tv.SearchResult, error) {
	shows, err := p.client.SearchTV(ctx, SearchTVParams{
		Query: query.Title,
		Year:  query.Year,
	})
	if err != nil {
		return nil, fmt.Errorf("tmdb search tv: %w", err)
	}

	results := make([]tv.SearchResult, 0, len(shows))
	for _, s := range shows {
		results = append(results, tv.SearchResult{
			ExternalID: core.NewExternalId(metadata.ProviderTMDBTV, strconv.Itoa(s.ID)),
			Title:      s.Name,
			Year:       yearFromDate(s.FirstAirDate),
			Overview:   s.Overview,
			Poster:     s.PosterPath,
			Popularity: s.Popularity,
		})
	}
	return results, nil
}

func (p *Provider) GetShow(ctx context.Context, id string) (*tv.ProviderShow, error) {
	tmdbID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tmdb tv id %q: %w", id, err)
	}

	d, err := p.client.GetTV(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("tmdb get tv %d: %w", tmdbID, err)
	}

	genres := make([]string, len(d.Genres))
	for i, g := range d.Genres {
		genres[i] = g.Name
	}

	networks := make([]string, len(d.Networks))
	for i, n := range d.Networks {
		networks[i] = n.Name
	}

	creators := make([]string, len(d.CreatedBy))
	for i, c := range d.CreatedBy {
		creators[i] = c.Name
	}

	firstAirDate, _ := parseDate(d.FirstAirDate)
	lastAirDate, _ := parseDate(d.LastAirDate)

	var runtime int
	if len(d.EpisodeRunTime) > 0 {
		runtime = d.EpisodeRunTime[0]
	}

	extIDs := p.tvExternalIDs(ctx, tmdbID)

	return &tv.ProviderShow{
		ExternalID:       core.NewExternalId(metadata.ProviderTMDBTV, strconv.Itoa(d.ID)),
		ExternalIDs:      extIDs,
		Title:            d.Name,
		OriginalTitle:    d.OriginalName,
		OriginalLanguage: d.OriginalLanguage,
		Overview:         d.Overview,
		Tagline:          d.Tagline,
		Status:           d.Status,
		FirstAirDate:     firstAirDate,
		LastAirDate:      lastAirDate,
		Year:             yearFromDate(d.FirstAirDate),
		Runtime:          runtime,
		SeasonCount:      d.NumberOfSeasons,
		EpisodeCount:     d.NumberOfEpisodes,
		Genres:           genres,
		OriginCountry:    d.OriginCountry,
		Networks:         networks,
		CreatedBy:        creators,
		Poster:           d.PosterPath,
		Backdrop:         d.BackdropPath,
		Rating:           float32(d.VoteAverage),
		VoteCount:        d.VoteCount,
		Popularity:       d.Popularity,
	}, nil
}

func (p *Provider) GetEpisodes(ctx context.Context, id string) ([]tv.ProviderEpisode, error) {
	tmdbID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tmdb tv id %q: %w", id, err)
	}

	show, err := p.client.GetTV(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("tmdb get tv %d (for episodes): %w", tmdbID, err)
	}

	var allEpisodes []tv.ProviderEpisode
	showExtID := core.NewExternalId(metadata.ProviderTMDBTV, strconv.Itoa(show.ID))

	for _, s := range show.Seasons {
		season, err := p.client.GetSeason(ctx, tmdbID, s.SeasonNumber)
		if err != nil {
			return nil, fmt.Errorf("tmdb get season %d: %w", s.SeasonNumber, err)
		}

		for _, ep := range season.Episodes {
			airDate, _ := parseDate(ep.AirDate)
			allEpisodes = append(allEpisodes, tv.ProviderEpisode{
				ExternalID:     core.NewExternalId(metadata.ProviderTMDBTV, strconv.Itoa(ep.ID)),
				ShowExternalID: showExtID,
				SeasonNumber:   ep.SeasonNumber,
				EpisodeNumber:  ep.EpisodeNumber,
				Title:          ep.Name,
				Overview:       ep.Overview,
				AirDate:        airDate,
				Runtime:        ep.Runtime,
				Still:          ep.StillPath,
				Rating:         float32(ep.VoteAverage),
				VoteCount:      ep.VoteCount,
				IsSeasonFinale: ep.EpisodeType == "finale",
			})
		}
	}

	return allEpisodes, nil
}

func (p *Provider) tvExternalIDs(ctx context.Context, tmdbID int) []core.ExternalId {
	ext, err := p.client.GetTVExternalIDs(ctx, tmdbID)
	if err != nil {
		return nil
	}
	return mapExternalIDs(ext)
}

func (p *Provider) movieExternalIDs(ctx context.Context, tmdbID int) []core.ExternalId {
	ext, err := p.client.GetMovieExternalIDs(ctx, tmdbID)
	if err != nil {
		return nil
	}
	return mapExternalIDs(ext)
}

func mapExternalIDs(e *ExternalIDs) []core.ExternalId {
	var out []core.ExternalId
	if e.IMDbID != "" {
		out = append(out, core.NewExternalId(metadata.ProviderIMDB, e.IMDbID))
	}
	if e.TVDBID != 0 {
		out = append(out, core.NewExternalId(metadata.ProviderTVDB, strconv.Itoa(e.TVDBID)))
	}
	if e.WikidataID != "" {
		out = append(out, core.NewExternalId(metadata.ProviderWikidata, e.WikidataID))
	}
	return out
}

func yearFromDate(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(date[:4])
	return year
}

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.Parse("2006-01-02", s)
}
