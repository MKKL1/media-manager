package domain

import "context"

// MediaRepository persists media entities.
type MediaRepository interface {
	Get(ctx context.Context, id MediaID) (*Media, error)
	GetByIdentity(ctx context.Context, id MediaIdentity) (*Media, error)
	Store(ctx context.Context, m *Media) error
	StoreWithItems(ctx context.Context, m *Media, items []MediaItem) error
	List(ctx context.Context, q MediaQuery) ([]Media, int, error)
}
