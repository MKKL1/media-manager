package tmdb

import (
	"context"
	"fmt"
	"server/internal/domain"
	"server/internal/media/movie"
	"server/internal/media/tv"
	"server/internal/metadata"
	"strconv"
	"time"
)

var (
	_ movie.Fetcher           = (*Provider)(nil)
	_ tv.Fetcher              = (*Provider)(nil)
	_ domain.ImageResolver    = (*Provider)(nil)
	_ metadata.SearchProvider = (*Provider)(nil)
)

type Provider struct {
	client *Client
}

func NewProvider(apiKey string) *Provider {
	return &Provider{client: NewClient(apiKey)}
}

func (p *Provider) FetchShow(ctx context.Context, id string) (*tv.ShowResult, error) {
	tmdbID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tmdb tv id %q: %w", id, err)
	}

	d, err := p.client.GetTV(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("tmdb get tv %d: %w", tmdbID, err)
	}

	extIDs := extractAppendedExternalIDs(d.ExternalIDs)

	seasons, err := p.resolveSeasons(ctx, tmdbID, d)
	if err != nil {
		return nil, fmt.Errorf("tmdb resolve seasons %d: %w", tmdbID, err)
	}

	var runtime *int
	if len(d.EpisodeRunTime) > 0 {
		runtime = new(d.EpisodeRunTime[0])
	}

	return &tv.ShowResult{
		Title:            d.Name,
		ExternalIDs:      extIDs,
		Seasons:          seasons,
		OriginalTitle:    new(d.OriginalName),
		OriginalLanguage: new(d.OriginalLanguage),
		Overview:         new(d.Overview),
		Tagline:          new(d.Tagline),
		Status:           new(d.Status),
		FirstAirDate:     parseOptionalDate(d.FirstAirDate),
		LastAirDate:      parseOptionalDate(d.LastAirDate),
		Runtime:          runtime,
		SeasonCount:      new(d.NumberOfSeasons),
		EpisodeCount:     new(d.NumberOfEpisodes),
		Genres:           mapGenreNames(d.Genres),
		Images:           collectImages(d.PosterPath, d.BackdropPath),
	}, nil
}

func (p *Provider) FetchMovie(ctx context.Context, id string) (*movie.MovieResult, error) {
	tmdbID, err := strconv.Atoi(id)
	if err != nil {
		return nil, fmt.Errorf("invalid tmdb movie id %q: %w", id, err)
	}

	d, err := p.client.GetMovie(ctx, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("tmdb get movie %d: %w", tmdbID, err)
	}

	extIDs := extractAppendedExternalIDs(d.ExternalIDs)

	return &movie.MovieResult{
		Title:            d.Title,
		ExternalIDs:      extIDs,
		OriginalTitle:    new(d.OriginalTitle),
		OriginalLanguage: new(d.OriginalLanguage),
		Overview:         new(d.Overview),
		Tagline:          new(d.Tagline),
		Status:           new(d.Status),
		ReleaseDate:      parseOptionalDate(d.ReleaseDate),
		Runtime:          new(d.Runtime),
		Genres:           mapGenreNames(d.Genres),
		Images:           collectImages(d.PosterPath, d.BackdropPath),
	}, nil
}

func (p *Provider) Search(ctx context.Context, query domain.SearchQuery) ([]domain.SearchResult, error) {
	hits, err := p.client.SearchMulti(ctx, query.Query, 1)
	if err != nil {
		return nil, fmt.Errorf("tmdb multi search: %w", err)
	}

	var results []domain.SearchResult
	for _, h := range hits {
		if r, ok := p.mapMultiResult(h); ok {
			results = append(results, r)
		}
	}
	return results, nil
}

func (p *Provider) mapMultiResult(h MultiSearchResult) (domain.SearchResult, bool) {
	switch h.MediaType {
	case "movie":
		return domain.SearchResult{
			PrimaryIdentity: domain.NewMediaIdentity(domain.KindTMDBMovie, strconv.Itoa(h.ID)),
			MediaType:       movie.MediaType,
			Title:           h.Title,
			Year:            yearFromDate(h.ReleaseDate),
			Overview:        h.Overview,
			Poster:          p.Resolve(h.PosterPath, domain.ImageQualityThumb, domain.ImageRolePoster),
			Popularity:      h.Popularity,
		}, true
	case "tv":
		return domain.SearchResult{
			PrimaryIdentity: domain.NewMediaIdentity(domain.KindTMDBTV, strconv.Itoa(h.ID)),
			MediaType:       tv.MediaType,
			Title:           h.Name,
			Year:            yearFromDate(h.FirstAirDate),
			Overview:        h.Overview,
			Poster:          p.Resolve(h.PosterPath, domain.ImageQualityThumb, domain.ImageRolePoster),
			Popularity:      h.Popularity,
		}, true
	default:
		return domain.SearchResult{}, false
	}
}

const episodeGroupTypeProduction = 6

func (p *Provider) resolveSeasons(ctx context.Context, tmdbID int, show *TVDetails) ([]tv.SeasonResult, error) {
	if seasons := p.tryAppendedEpisodeGroups(ctx, show.EpisodeGroups); len(seasons) > 0 {
		return seasons, nil
	}
	return p.fetchRegularSeasons(ctx, tmdbID, show)
}

func (p *Provider) tryAppendedEpisodeGroups(ctx context.Context, groups *EpisodeGroupsResult) []tv.SeasonResult {
	if groups == nil || len(groups.Results) == 0 {
		return nil
	}

	var best *EpisodeGroupSummary
	for i := range groups.Results {
		if groups.Results[i].Type == episodeGroupTypeProduction {
			best = &groups.Results[i]
			break
		}
	}
	if best == nil {
		return nil
	}

	detail, err := p.client.GetEpisodeGroupDetail(ctx, best.ID)
	if err != nil {
		return nil // fall through to regular seasons
	}

	seasons := make([]tv.SeasonResult, 0, len(detail.Groups))
	for _, g := range detail.Groups {
		seasons = append(seasons, tv.SeasonResult{
			Number:   g.Order,
			Title:    new(g.Name),
			Episodes: mapEpisodes(g.Episodes),
		})
	}
	return seasons
}

func (p *Provider) fetchRegularSeasons(ctx context.Context, tmdbID int, show *TVDetails) ([]tv.SeasonResult, error) {
	results := make([]tv.SeasonResult, 0, len(show.Seasons))
	for _, s := range show.Seasons {
		season, err := p.client.GetSeason(ctx, tmdbID, s.SeasonNumber)
		if err != nil {
			return nil, fmt.Errorf("get season %d: %w", s.SeasonNumber, err)
		}
		results = append(results, tv.SeasonResult{
			Number:   season.SeasonNumber,
			Title:    new(season.Name),
			Episodes: mapEpisodes(season.Episodes),
		})
	}
	return results, nil
}

func extractAppendedExternalIDs(ext *ExternalIDs) []domain.MediaIdentity {
	if ext == nil {
		return nil
	}
	return mapExternalIDs(ext)
}

func mapExternalIDs(e *ExternalIDs) []domain.MediaIdentity {
	var out []domain.MediaIdentity
	if e.IMDbID != "" {
		out = append(out, domain.NewMediaIdentity(domain.KindIMDB, e.IMDbID))
	}
	if e.TVDBID != 0 {
		out = append(out, domain.NewMediaIdentity(domain.KindTVDB, strconv.Itoa(e.TVDBID)))
	}
	if e.WikidataID != "" {
		out = append(out, domain.NewMediaIdentity(domain.KindWikidata, e.WikidataID))
	}
	return out
}

func mapEpisodes(eps []Episode) []tv.EpisodeResult {
	out := make([]tv.EpisodeResult, 0, len(eps))
	for _, ep := range eps {
		out = append(out, tv.EpisodeResult{
			ExternalID:    domain.NewMediaIdentity(domain.KindTMDBEpisode, strconv.Itoa(ep.ID)),
			SeasonNumber:  ep.SeasonNumber,
			EpisodeNumber: ep.EpisodeNumber,
			Title:         ep.Name,
			Overview:      optStr(ep.Overview),
			AirDate:       parseOptionalDate(ep.AirDate),
			Runtime:       optInt(ep.Runtime),
			Still:         optStr(ep.StillPath),
			IsFinale:      ep.EpisodeType == "finale",
		})
	}
	return out
}

func mapGenreNames(genres []Genre) []string {
	out := make([]string, len(genres))
	for i, g := range genres {
		out[i] = g.Name
	}
	return out
}

func collectImages(poster, backdrop string) []domain.ProviderImage {
	var out []domain.ProviderImage
	if poster != "" {
		out = append(out, domain.ProviderImage{Role: domain.ImageRolePoster, Path: poster})
	}
	if backdrop != "" {
		out = append(out, domain.ProviderImage{Role: domain.ImageRoleBackdrop, Path: backdrop})
	}
	return out
}

func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optInt(i int) *int {
	if i == 0 {
		return nil
	}
	return &i
}

func parseOptionalDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func yearFromDate(date string) int {
	if len(date) < 4 {
		return 0
	}
	year, _ := strconv.Atoi(date[:4])
	return year
}
