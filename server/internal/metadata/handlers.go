package metadata

import (
	"context"
	"fmt"
	"server/internal/core"
)

type MediaTypeHandler interface {
	// FetchMedia Decides what provider to query and maps to generic media type (Query only)
	FetchMedia(ctx context.Context, id core.ExternalId) (*core.MediaWithItems, error)
}

type Handlers map[core.MediaType]MediaTypeHandler

func (h Handlers) Get(t core.MediaType) (MediaTypeHandler, error) {
	handler, ok := h[t]
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q: %w", t, core.ErrInvalidInput)
	}
	return handler, nil
}
