package metadata

import (
	"context"
	"server/internal/core"
)

type MediaRepository interface {
	Get(ctx context.Context, id core.MediaId) (*core.Media, error)
	StoreMediaWithItems(ctx context.Context, m core.MediaWithItems) error
	GetByExternalID(ctx context.Context, id core.ExternalId) (*core.Media, error)
	GetMediaIdByExternalId(ctx context.Context, id core.ExternalId) (*core.MediaId, error)
}
