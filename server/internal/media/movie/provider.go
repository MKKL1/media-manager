package movie

import "context"

type Fetcher interface {
	FetchMovie(ctx context.Context, id string) (*MovieResult, error)
}
