package pkg

import (
	"context"
	"database/sql"
	"server/pkg/domain"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	Type    domain.MediaType
	Handler MediaHandler
	Mount   func(router chi.Router)
	Migrate func(db *sql.DB)
}

// MediaHandler Every media type module must implement this
// Implemented by movie.Handler, tv.Handler.
type MediaHandler interface {
	Type() domain.MediaType
	// FetchMedia calls external metadata provider
	FetchMedia(ctx context.Context, id domain.MediaIdentity) (*domain.MediaWithItems, error)
	ToSummary(media domain.Media) (domain.MediaSummary, error)
}
