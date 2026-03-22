package tmdb

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"server/internal/core"
	"strconv"
	"time"

	"github.com/goccy/go-json"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/sony/gobreaker"
)

//TODO cache, but how?
//in app? lost on restart
//redis? lost on restart, adds complexity, most often app will be restarted with redis
//postgres? is it good for that? won't cause too many writes to storage?

type Client struct {
	http    *http.Client
	breaker *gobreaker.CircuitBreaker
	apiKey  string
	baseURL string
}

func NewClient(apiKey string) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	retryClient.HTTPClient = &http.Client{Timeout: 5 * time.Second}

	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Interval: 60 * time.Second,
		Timeout:  30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})

	return &Client{
		http:    retryClient.StandardClient(),
		breaker: breaker,
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

func (c *Client) GetTV(ctx context.Context, id int) (*TVDetails, error) {
	data, err := c.get(ctx, "/3/tv/"+strconv.Itoa(id), url.Values{})
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
	data, err := c.get(ctx, "/3/movie/"+strconv.Itoa(id), url.Values{})
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

		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, fmt.Errorf("tmdb api: %w", core.ErrRateLimited)
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
		}

		return io.ReadAll(resp.Body)
	})
	if err != nil {
		return nil, err
	}

	return result.([]byte), nil
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
