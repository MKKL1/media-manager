package tmdb

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"server/internal/domain"
	"strconv"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/maypok86/otter"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

//TODO cache, but how?
//in app? lost on restart
//redis? lost on restart, adds complexity, most often app will be restarted with redis
//postgres? is it good for that? won't cause too many writes to storage?

type Client struct {
	http    *http.Client
	breaker *gobreaker.CircuitBreaker
	cache   otter.Cache[string, []byte]
	apiKey  string
	baseURL string
}

func NewClient(apiKey string) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	retryClient.HTTPClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: otelhttp.NewTransport(http.DefaultTransport,
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				return "tmdb " + r.Method + " " + r.URL.Path
			}),
			otelhttp.WithSpanOptions(trace.WithAttributes(
				attribute.String("plugin", "tmdb"),
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

	cache, err := otter.MustBuilder[string, []byte](5000).
		WithTTL(5 * time.Minute).
		Build()
	if err != nil {
		panic("failed to build tmdb cache: " + err.Error())
	}

	return &Client{
		http:    retryClient.StandardClient(),
		breaker: breaker,
		cache:   cache,
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org",
	}
}
func (c *Client) SearchTV(ctx context.Context, params SearchTVParams) ([]TVShow, error) {
	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	q := url.Values{}
	q.Set("query", params.Query)
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}
	if params.Year > 0 {
		q.Set("year", strconv.Itoa(params.Year))
	}

	data, err := c.get(ctx, "/3/search/tv", q)
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []TVShow `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (c *Client) SearchMovies(ctx context.Context, params SearchMovieParams) ([]Movie, error) {
	if params.Query == "" {
		return nil, fmt.Errorf("query is required")
	}

	q := url.Values{}
	q.Set("query", params.Query)
	if params.Page > 0 {
		q.Set("page", strconv.Itoa(params.Page))
	}
	if params.Year > 0 {
		q.Set("year", strconv.Itoa(params.Year))
	}

	data, err := c.get(ctx, "/3/search/movie", q)
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []Movie `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (c *Client) SearchMulti(ctx context.Context, query string, page int) ([]MultiSearchResult, error) {
	q := url.Values{}
	q.Set("query", query)
	if page > 0 {
		q.Set("page", strconv.Itoa(page))
	}

	data, err := c.get(ctx, "/3/search/multi", q)
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []MultiSearchResult `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return response.Results, nil
}

func (c *Client) GetTV(ctx context.Context, id int) (*TVDetails, error) {
	q := url.Values{}
	q.Set("append_to_response", "external_ids,episode_groups")

	data, err := c.get(ctx, "/3/tv/"+strconv.Itoa(id), q)
	if err != nil {
		return nil, err
	}

	var details TVDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}

	return &details, nil
}

func (c *Client) GetMovie(ctx context.Context, id int) (*MovieDetails, error) {
	q := url.Values{}
	q.Set("append_to_response", "external_ids")

	data, err := c.get(ctx, "/3/movie/"+strconv.Itoa(id), q)
	if err != nil {
		return nil, err
	}

	var details MovieDetails
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}

	return &details, nil
}

func (c *Client) get(ctx context.Context, path string, params url.Values) ([]byte, error) {
	key := path + "?" + params.Encode()

	if data, ok := c.cache.Get(key); ok {
		return data, nil
	}

	result, err := c.breaker.Execute(func() (interface{}, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path+"?"+params.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, tmdbStatusError(resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	})
	if err != nil {
		return nil, err
	}

	data := result.([]byte)
	c.cache.Set(key, data)

	return data, nil
}

func tmdbStatusError(status int) error {
	switch status {
	case http.StatusTooManyRequests:
		return fmt.Errorf("tmdb: %w", domain.ErrRateLimited)
	case http.StatusNotFound:
		return domain.Permanent(fmt.Errorf("tmdb: %w", domain.ErrNotFound)) //TODO i don't like this, it's not a way in which golang should handle errors
	case http.StatusUnauthorized, http.StatusForbidden:
		return domain.Permanent(fmt.Errorf("tmdb: access denied (status %d)", status))
	default:
		if status >= 400 && status < 500 {
			return domain.Permanent(fmt.Errorf("tmdb: client error (status %d)", status))
		}
		return fmt.Errorf("tmdb: server error (status %d)", status)
	}
}

func (c *Client) GetSeason(ctx context.Context, tvID int, seasonNumber int) (*Season, error) {
	data, err := c.get(ctx, fmt.Sprintf("/3/tv/%d/season/%d", tvID, seasonNumber), url.Values{})
	if err != nil {
		return nil, err
	}

	var season Season
	if err := json.Unmarshal(data, &season); err != nil {
		return nil, err
	}

	return &season, nil
}

func (c *Client) GetTVExternalIDs(ctx context.Context, id int) (*ExternalIDs, error) {
	data, err := c.get(ctx, fmt.Sprintf("/3/tv/%d/external_ids", id), url.Values{})
	if err != nil {
		return nil, err
	}

	var ext ExternalIDs
	if err := json.Unmarshal(data, &ext); err != nil {
		return nil, err
	}

	return &ext, nil
}

func (c *Client) GetMovieExternalIDs(ctx context.Context, id int) (*ExternalIDs, error) {
	data, err := c.get(ctx, fmt.Sprintf("/3/movie/%d/external_ids", id), url.Values{})
	if err != nil {
		return nil, err
	}

	var ext ExternalIDs
	if err := json.Unmarshal(data, &ext); err != nil {
		return nil, err
	}

	return &ext, nil
}

//TODO this should probably not be exposed to other parts of the system
// Exposing this means that it's hard to optimize media fetching
// Maybe it would make more sense if users could choose some episode system when fetching media

func (c *Client) GetEpisodeGroups(ctx context.Context, seriesID int) ([]EpisodeGroupSummary, error) {
	data, err := c.get(ctx, fmt.Sprintf("/3/tv/%d/episode_groups", seriesID), url.Values{})
	if err != nil {
		return nil, err
	}

	var response struct {
		Results []EpisodeGroupSummary `json:"results"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}
	return response.Results, nil
}

func (c *Client) GetEpisodeGroupDetail(ctx context.Context, groupID string) (*EpisodeGroupDetailResponse, error) {
	data, err := c.get(ctx, "/3/tv/episode_group/"+groupID, url.Values{})
	if err != nil {
		return nil, err
	}

	var detail EpisodeGroupDetailResponse
	if err := json.Unmarshal(data, &detail); err != nil {
		return nil, err
	}
	return &detail, nil
}
