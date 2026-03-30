package anime_list

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"server/internal/domain"
	"server/internal/metadata"
	"strconv"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ metadata.MappingSource = (*Source)(nil)

type Source struct {
	client  *http.Client
	breaker *gobreaker.CircuitBreaker
	url     string
}

func NewSource(url string) *Source {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	retryClient.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return "anime-list " + r.Method + " " + r.URL.Path
			}),
			otelhttp.WithSpanOptions(trace.WithAttributes(
				attribute.String("plugin", "anime-list"),
				attribute.String("component", "plugin"),
			)),
		),
	}

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Interval: 60 * time.Second,
		Timeout:  30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	return &Source{
		client:  retryClient.StandardClient(),
		breaker: breaker,
		url:     "https://raw.githubusercontent.com/Anime-Lists/anime-lists/refs/heads/master/anime-list-full.xml",
	}
}

func (p Source) Name() string {
	return "github.com/Anime-Lists/anime-lists"
}

func (p Source) Load(ctx context.Context, lastVersion string) (*metadata.MappingData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return nil, err
	}

	if lastVersion != "" {
		req.Header.Set("If-None-Match", lastVersion)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, nil // unchanged
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var list xmlAnimeList
	if err := xml.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}

	entries := make([]metadata.MappingEntry, 0, len(list.Anime))
	for _, a := range list.Anime {
		entries = append(entries, a.toMappingEntry())
	}

	return &metadata.MappingData{
		Version: resp.Header.Get("ETag"),
		Entries: entries,
	}, nil
}
func (a *xmlAnime) toMappingEntry() metadata.MappingEntry {
	anidbID := domain.NewMediaIdentity(domain.KindAniDB, itoa(a.AniDBID))

	entry := metadata.MappingEntry{
		IDs: []domain.MediaIdentity{anidbID},
	}

	// tvdbid="movie" means no tvdb series
	if a.TVDBID != "" && a.TVDBID != "movie" {
		entry.IDs = append(entry.IDs, domain.NewMediaIdentity(domain.KindTVDB, a.TVDBID))
	}

	// tmdbtv = tmdb TV series ID
	if a.TMDBTv != "" {
		entry.IDs = append(entry.IDs, domain.NewMediaIdentity(domain.KindTMDBTV, a.TMDBTv))
	}

	// tmdbid = tmdb movie ID
	if a.TMDBId != "" {
		entry.IDs = append(entry.IDs, domain.NewMediaIdentity(domain.KindTMDBMovie, a.TMDBId))
	}

	if a.IMDBId != "" {
		entry.IDs = append(entry.IDs, domain.NewMediaIdentity(domain.KindIMDB, a.IMDBId))
	}

	// Season mappings — "a" means all seasons, skip those
	if season, err := strconv.Atoi(a.DefaultTVDBSeason); err == nil {
		entry.Seasons = append(entry.Seasons, metadata.SeasonMapping{
			Provider:     string(domain.KindTVDB.ProviderName), //TODO not sure how to do it here
			SeasonNumber: season,
		})
	}

	if season, err := strconv.Atoi(a.TMDBSeason); err == nil {
		entry.Seasons = append(entry.Seasons, metadata.SeasonMapping{
			Provider:     string(domain.KindTMDBTV.ProviderName),
			SeasonNumber: season,
		})
	}

	return entry
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
