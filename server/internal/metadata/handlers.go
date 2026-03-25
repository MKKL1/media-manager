package metadata

import (
	"fmt"
	"server/internal/domain"
)

type Handlers map[domain.MediaType]MediaHandler

func (h Handlers) Get(t domain.MediaType) (MediaHandler, error) {
	handler, ok := h[t]
	if !ok {
		return nil, fmt.Errorf("unsupported media type %q: %w", t, domain.ErrInvalidInput)
	}
	return handler, nil
}
