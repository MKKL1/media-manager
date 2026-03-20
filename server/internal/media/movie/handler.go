package movie

import (
	"context"
	"fmt"
	"server/internal/core"
	"server/internal/metadata"

	"github.com/google/uuid"
)

var _ metadata.MediaTypeHandler = (*Handler)(nil)

type Handler struct {
	fetchers map[string]Fetcher
}

func NewMovieHandler(fetchers map[string]Fetcher) *Handler {
	return &Handler{fetchers: fetchers}
}

// FetchMedia Decides what provider to query and maps to generic media type
func (h *Handler) FetchMedia(ctx context.Context, id core.ExternalId) (*core.MediaWithItems, error) {
	fetcher, ok := h.fetchers[id.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found for movie", id.Provider)
	}

	movie, err := fetcher.GetMovie(ctx, id.Id)
	if err != nil {
		return nil, err
	}

	movieMeta := Metadata{
		OriginalTitle: movie.OriginalTitle,
		Overview:      movie.Overview,
		Tagline:       movie.Tagline,
		ReleaseDate:   movie.ReleaseDate,
		Runtime:       movie.Runtime,
		Genres:        movie.Genres,
		Poster:        movie.Poster,
		Backdrop:      movie.Backdrop,
	}

	mediaID := core.GenerateMediaID()

	externalIds := []core.ExternalId{id}
	externalIds = append(externalIds, movie.ExternalIDs...)

	media := core.Media{
		ID:                mediaID,
		Type:              string(MediaType),
		Title:             movie.Title,
		Status:            movie.Status,
		PrimaryExternalId: id,
		ExternalIds:       externalIds,
		Metadata:          movieMeta,
	}

	// For movies, the item array just contains the movie itself as a single item
	movieItem := core.MediaItem{
		ID:      uuid.New(),
		MediaId: mediaID,
		Status:  "Unknown",
	}

	return &core.MediaWithItems{
		Media: media,
		Items: []core.MediaItem{movieItem},
	}, nil
}
