package tv

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

func NewTVHandler(fetchers map[string]Fetcher) *Handler {
	return &Handler{fetchers: fetchers}
}

// FetchMedia Decides what provider to query and maps to generic media type
func (h *Handler) FetchMedia(ctx context.Context, id core.ExternalId) (*core.MediaWithItems, error) {
	fetcher, ok := h.fetchers[id.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found for tv", id.Provider)
	}

	show, err := fetcher.GetShow(ctx, id.Id)
	if err != nil {
		return nil, err
	}

	episodes, err := fetcher.GetEpisodes(ctx, id.Id)
	if err != nil {
		return nil, err
	}

	mediaID := core.GenerateMediaID()

	seasonsMap := make(map[int][]EpisodeInfo)
	var items []core.MediaItem

	for _, ep := range episodes {
		itemID := uuid.New()

		epMeta := EpisodeMetadata{
			OriginalTitle: ep.Title,
			Overview:      ep.Overview,
			AirDate:       ep.AirDate,
			Runtime:       ep.Runtime,
			Still:         ep.Still,
			SeasonNumber:  ep.SeasonNumber,
			EpisodeNumber: ep.EpisodeNumber,
		}

		items = append(items, core.MediaItem{
			ID:       itemID,
			MediaId:  mediaID,
			Status:   "Unknown",
			Metadata: epMeta,
		})

		seasonsMap[ep.SeasonNumber] = append(seasonsMap[ep.SeasonNumber], EpisodeInfo{
			EpisodeNumber: ep.EpisodeNumber,
			ID:            itemID,
		})
	}

	var seasons []SeasonMetadata
	for sNum, eps := range seasonsMap {
		seasons = append(seasons, SeasonMetadata{
			SeasonNumber: sNum,
			Episodes:     eps,
		})
	}

	tvMeta := Metadata{
		OriginalTitle: show.OriginalTitle,
		Overview:      show.Overview,
		Tagline:       show.Tagline,
		FirstAirDate:  show.FirstAirDate,
		LastAirDate:   show.LastAirDate,
		SeasonCount:   show.SeasonCount,
		EpisodeCount:  show.EpisodeCount,
		Runtime:       show.Runtime,
		Genres:        show.Genres,
		Poster:        show.Poster,
		Backdrop:      show.Backdrop,
		Seasons:       seasons,
	}

	externalIds := []core.ExternalId{id}
	externalIds = append(externalIds, show.ExternalIDs...)

	media := core.Media{
		ID:                mediaID,
		Type:              string(MediaType),
		Title:             show.Title,
		Status:            show.Status,
		PrimaryExternalId: id,
		ExternalIds:       externalIds,
		Metadata:          tvMeta,
	}

	return &core.MediaWithItems{
		Media: media,
		Items: items,
	}, nil
}
