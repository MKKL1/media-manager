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
	fetchers         map[string]Fetcher
	seasonMappingSrv AnimeMappingService
}

func NewTVHandler(fetchers map[string]Fetcher) *Handler {
	return &Handler{fetchers: fetchers}
}

// FetchMedia Decides what provider to query and maps to generic media type
func (h *Handler) FetchMedia(ctx context.Context, id core.ExternalId) (*core.MediaWithItems, error) {

	//TODO estimate episode-season mapping here

	fetcher, ok := h.fetchers[id.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %s not found for tv", id.Provider)
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

	mediaID := core.GenerateMediaID()

	var seasonData []SeasonMetadata
	var items []core.MediaItem

	for _, se := range seasons {
		itemID := uuid.New()
		var epInfo []EpisodeInfo

		for _, ep := range se.Episodes {
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

func fetchEpisodes(ctx context.Context, id core.ExternalId, fetcher Fetcher) ([]ProviderSeason, error) {
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

	episodes, err := fetcher.GetEpisodes(ctx, id.Id)
	if err != nil {
		return nil, err
	}

	return episodes, nil
}

func fetchEpisodesFromGroup(ctx context.Context, id core.ExternalId, gf EpisodeGroupFetcher) ([]ProviderSeason, error) {
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
			ExternalID:   new(core.NewExternalId(metadata.ProviderTMDBSeason, g.ID)),
			Title:        new(g.Name),
			Episodes:     g.Episodes,
		})
		//TODO here season-episode mapping doesn't get applied
	}
	return out
}
