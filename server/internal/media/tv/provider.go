package tv

import "context"

type Fetcher interface {
	FetchShow(ctx context.Context, id string) (*ShowResult, error)
}
