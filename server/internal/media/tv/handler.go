package tv

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

func NewTVHandler(fetchers map[string]Fetcher) *Handler {
	return &Handler{fetchers: fetchers}
}

func (h *Handler) FetchMedia(ctx context.Context, id domain.ExternalId) (*domain.MediaWithItems, error) {
	fetcher, ok := h.fetchers[id.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found for tv: %w", id.Provider, domain.ErrNoProvider)
	}

	//This can be easily optimized to be one call
	show, err := fetcher.GetShow(ctx, id.Id)
	if err != nil {
		return nil, err
	}

	seasons, err := fetchEpisodes(ctx, id, fetcher)
	if err != nil {
		return nil, err
	}

	mediaID := domain.GenerateMediaID()

	var seasonData []SeasonMetadata
	var items []domain.MediaItem

	for _, se := range seasons {
		var epInfo []EpisodeInfo

		for _, ep := range se.Episodes {
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

			items = append(items, domain.MediaItem{
				ID:       itemID,
				MediaId:  mediaID,
				Status:   "Unknown",
				Metadata: epMeta,
			})

			epInfo = append(epInfo, EpisodeInfo{
				EpisodeNumber: ep.EpisodeNumber,
				ID:            itemID,
			})
		}

		seasonData = append(seasonData, SeasonMetadata{
			SeasonNumber: se.SeasonNumber,
			Episodes:     epInfo,
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
		Seasons:       seasonData,
	}

	externalIds := []domain.ExternalId{id}
	externalIds = append(externalIds, show.ExternalIDs...)

	media := domain.Media{
		ID:                mediaID,
		Type:              string(MediaType),
		Title:             show.Title,
		Status:            show.Status,
		PrimaryExternalId: id,
		ExternalIds:       externalIds,
		Metadata:          tvMeta,
	}

	return &domain.MediaWithItems{
		Media: media,
		Items: items,
	}, nil
}

func fetchEpisodes(ctx context.Context, id domain.ExternalId, fetcher Fetcher) ([]ProviderSeason, error) {
	gf, ok := fetcher.(EpisodeGroupFetcher)
	if ok {
		episodes, err := fetchEpisodesFromGroup(ctx, id, gf)
		if err != nil {
			return nil, err
		}
		// TODO Here it would be very useful to somehow verify that it's correct, for example we may get 1 episode, but there are 20
		if len(episodes) != 0 {
			return episodes, nil
		}
	}

	return fetcher.GetEpisodes(ctx, id.Id)
}

func fetchEpisodesFromGroup(ctx context.Context, id domain.ExternalId, gf EpisodeGroupFetcher) ([]ProviderSeason, error) {
	groups, err := gf.GetEpisodeGroups(ctx, id.Id)
	if err != nil {
		return nil, err
	}
	if best := selectBestEpisodeGroup(groups); best != nil {
		detail, err := gf.GetEpisodeGroupDetail(ctx, best.ID)
		if err == nil {
			return flattenEpisodeGroupDetail(detail), nil
		}
	}
	return nil, nil
}

func selectBestEpisodeGroup(groups []EpisodeGroup) *EpisodeGroup {
	for _, g := range groups {
		if g.Type == EpisodeGroupTypeProduction {
			return &g
		}
	}
	return nil
}

func flattenEpisodeGroupDetail(detail *EpisodeGroupDetail) []ProviderSeason {
	var out []ProviderSeason
	for _, g := range detail.Groups {
		out = append(out, ProviderSeason{
			SeasonNumber: g.Order,
			ExternalID:   new(domain.NewExternalId(domain.ProviderTMDBSeason, g.ID)),
			Title:        new(g.Name),
			Episodes:     g.Episodes,
		})
	}
	return out
}
