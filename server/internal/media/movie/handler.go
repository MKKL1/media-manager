package movie

import (
	"context"
	"fmt"
	"server/internal/domain"
	"server/internal/metadata"

	"github.com/google/uuid"
)

var _ metadata.MediaHandler = (*Handler)(nil)

type Handler struct {
	fetchers map[string]Fetcher
}

func (h *Handler) Type() domain.MediaType {
	return MediaType
}

func (h *Handler) ToSummary(media domain.Media) (domain.MediaSummary, error) {
	//TODO implement me
	panic("implement me")
}

func NewMovieHandler(fetchers map[string]Fetcher) *Handler {
	return &Handler{fetchers: fetchers}
}

// FetchMedia Decides what provider to query and maps to generic media type
func (h *Handler) FetchMedia(ctx context.Context, id domain.ExternalId) (*domain.MediaWithItems, error) {
	fetcher, ok := h.fetchers[id.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found for movie", id.Provider)
	}

	movie, err := fetcher.GetMovie(ctx, id.Id)
	if err != nil {
		return nil, err
	}

	_ = Metadata{
		OriginalTitle: movie.OriginalTitle,
		Overview:      movie.Overview,
		Tagline:       movie.Tagline,
		ReleaseDate:   movie.ReleaseDate,
		Runtime:       movie.Runtime,
		Genres:        movie.Genres,
		Poster:        movie.Poster,
		Backdrop:      movie.Backdrop,
	}

	mediaID := domain.GenerateMediaID()

	externalIds := []domain.ExternalId{id}
	externalIds = append(externalIds, movie.ExternalIDs...)

	media := domain.Media{
		ID:                mediaID,
		Type:              MediaType,
		Title:             movie.Title,
		Status:            movie.Status,
		PrimaryExternalId: id,
		ExternalIds:       externalIds,
		Metadata:          nil, //TODO
	}

	// For movies, the item array just contains the movie itself as a single item
	movieItem := domain.MediaItem{
		ID:      uuid.New(),
		MediaId: mediaID,
		Status:  "Unknown",
	}

	return &domain.MediaWithItems{
		Media: media,
		Items: []domain.MediaItem{movieItem},
	}, nil
}
