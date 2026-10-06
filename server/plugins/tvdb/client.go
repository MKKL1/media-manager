package tvdb

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/maypok86/otter"
	"github.com/sony/gobreaker"
)

var ErrNotFound = errors.New("tvdb: not found")

type Client struct {
	http    *http.Client
	breaker *gobreaker.CircuitBreaker
	cache   otter.Cache[string, []byte]
	apiKey  string
	pin     string
	baseURL string

	tokenLock sync.Mutex
	token     string
}

func NewClient(apiKey, pin string) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.Logger = nil
	retryClient.HTTPClient = &http.Client{
		Timeout: 10 * time.Second,
	}
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Interval: 60 * time.Second,
		Timeout:  30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	})
	cache, err := otter.MustBuilder[string, []byte](5000).WithTTL(5 * time.Minute).Build()
	if err != nil {
		panic("failed to build tvdb cache: " + err.Error())
	}
	return &Client{
		http: retryClient.StandardClient(), breaker: breaker, cache: cache,
		apiKey: apiKey, pin: pin, baseURL: "https://api4.thetvdb.com/v4",
	}
}

func (c *Client) GetSeries(ctx context.Context, id int) (*Series, error) {
	var s Series
	err := c.getData(ctx, fmt.Sprintf("/series/%d/extended", id), url.Values{"short": {"true"}}, &s)
	return &s, err
}

func (c *Client) GetSeriesTranslation(ctx context.Context, id int, language string) (*Translation, error) {
	var t Translation
	err := c.getData(ctx, fmt.Sprintf("/series/%d/translations/%s", id, language), nil, &t)
	return &t, err
}

func (c *Client) GetEpisodes(ctx context.Context, id int, seasonType, language string) ([]Episode, error) {
	var all []Episode
	for page := 0; ; page++ {
		var data struct {
			Episodes []Episode `json:"episodes"`
		}
		q := url.Values{"page": {strconv.Itoa(page)}}
		next, err := c.get(ctx, fmt.Sprintf("/series/%d/episodes/%s/%s", id, seasonType, language), q, &data)
		if err != nil {
			return nil, fmt.Errorf("episodes page %d: %w", page, err)
		}
		all = append(all, data.Episodes...)
		if next == "" {
			return all, nil
		}
	}
}

func (c *Client) GetMovie(ctx context.Context, id int) (*Movie, error) {
	var m Movie
	err := c.getData(ctx, fmt.Sprintf("/movies/%d/extended", id), url.Values{"short": {"true"}}, &m)
	return &m, err
}

func (c *Client) GetMovieTranslation(ctx context.Context, id int, language string) (*Translation, error) {
	var t Translation
	err := c.getData(ctx, fmt.Sprintf("/movies/%d/translations/%s", id, language), nil, &t)
	return &t, err
}

func (c *Client) getData(ctx context.Context, path string, q url.Values, data any) error {
	_, err := c.get(ctx, path, q, data)
	return err
}

func (c *Client) get(ctx context.Context, path string, q url.Values, data any) (next string, err error) {
	key := path + "?" + q.Encode()
	body, ok := c.cache.Get(key)
	if !ok {
		status := 0
		for attempt := 0; attempt < 2; attempt++ {
			if status, body, err = c.request(ctx, key); err != nil {
				return "", err
			}
			if status != http.StatusUnauthorized {
				break
			}
			c.forgetToken()
		}
		switch {
		case status == http.StatusNotFound:
			return "", ErrNotFound
		case status != http.StatusOK:
			return "", fmt.Errorf("tvdb: status %d", status)
		}
		c.cache.Set(key, body)
	}
	var answer struct {
		Data  json.NoCopyRawMessage `json:"data"`
		Links struct {
			Next string `json:"next"`
		} `json:"links"`
	}
	if err := json.Unmarshal(body, &answer); err != nil {
		return "", err
	}
	if err := json.Unmarshal(answer.Data, data); err != nil {
		return "", err
	}
	return answer.Links.Next, nil
}

func (c *Client) request(ctx context.Context, pathAndQuery string) (int, []byte, error) {
	token, err := c.login(ctx)
	if err != nil {
		return 0, nil, err
	}
	type reply struct {
		status int
		body   []byte
	}
	r, err := c.breaker.Execute(func() (any, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+pathAndQuery, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("tvdb: status %d", resp.StatusCode)
		}
		return reply{resp.StatusCode, b}, nil
	})
	if err != nil {
		return 0, nil, err
	}
	return r.(reply).status, r.(reply).body, nil
}

func (c *Client) login(ctx context.Context) (string, error) {
	c.tokenLock.Lock()
	defer c.tokenLock.Unlock()
	if c.token != "" {
		return c.token, nil
	}
	login := map[string]string{"apikey": c.apiKey}
	if c.pin != "" {
		login["pin"] = c.pin
	}
	payload, err := json.Marshal(login)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/login", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tvdb: login: status %d", resp.StatusCode)
	}
	var answer struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if err := json.Unmarshal(b, &answer); err != nil {
		return "", err
	}
	if answer.Data.Token == "" {
		return "", errors.New("tvdb: login gave no token")
	}
	c.token = answer.Data.Token
	return c.token, nil
}

func (c *Client) forgetToken() {
	c.tokenLock.Lock()
	c.token = ""
	c.tokenLock.Unlock()
}
