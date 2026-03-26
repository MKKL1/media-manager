package tv

import (
	"context"
	"fmt"
	"server/internal/domain"
	"server/internal/metadata"
	"time"

	"github.com/goccy/go-json"
	"github.com/google/uuid"
)

const summaryMaxLength = 150

var _ metadata.MediaHandler = (*Handler)(nil)

type Handler struct {
	fetchers       map[string]Fetcher
	imageResolvers map[string]domain.ImageResolver
}

func NewTVHandler(fetchers map[string]Fetcher, imageResolvers map[string]domain.ImageResolver) *Handler {
	return &Handler{fetchers: fetchers, imageResolvers: imageResolvers}
}

func (h *Handler) Type() domain.MediaType {
	return MediaType
}

func (h *Handler) ToSummary(media domain.Media) (domain.MediaSummary, error) {
	var meta Metadata

	if len(media.Metadata) > 0 {
		if err := json.Unmarshal(media.Metadata, &meta); err != nil {
			return domain.MediaSummary{}, fmt.Errorf("json.Unmarshal: %w", err)
		}
	}

	ires, ok := h.imageResolvers[media.PrimaryExternalId.Provider]
	if !ok {
		return domain.MediaSummary{}, fmt.Errorf("h.imageResolvers %s not found for tv: %w", media.PrimaryExternalId.Provider, domain.ErrNoProvider)
	}

	var img domain.Image
	var found = false
	for _, e := range media.Images {
		if e.Role == domain.ImageRolePoster {
			img = e
			found = true
			break
		}
	}
	var posterPath = ""
	if found {
		posterPath = ires.Resolve(img, domain.ImageQualityThumb)
	}
	return domain.MediaSummary{
		Id:            media.ID,
		Type:          media.Type,
		Title:         media.Title,
		OriginalTitle: meta.OriginalTitle,
		OriginalLang:  "",
		Monitored:     media.Monitored,
		Status:        media.Status,
		Summary:       shorten(meta.Overview, summaryMaxLength) + "...",
		ReleaseDate:   meta.FirstAirDate,
		Source:        media.PrimaryExternalId,
		PosterPath:    posterPath,
		Metadata:      nil,
	}, nil
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
		var epCount = 0
		var releasedEpCount = 0

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

			epMetaJSON, err := json.Marshal(epMeta)
			if err != nil {
				return nil, fmt.Errorf("json.Marshal epMeta: %w", err)
			}

			items = append(items, domain.MediaItem{
				ID:       itemID,
				MediaId:  mediaID,
				Status:   "Unknown",
				Metadata: epMetaJSON,
			})

			epCount++
			//TODO get actual data for that
			releasedEpCount++
		}

		seasonData = append(seasonData, SeasonMetadata{
			SeasonNumber:         se.SeasonNumber,
			EpisodeCount:         epCount,
			EpsiodeReleasedCount: releasedEpCount,
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

	tvMetaJSON, err := json.Marshal(tvMeta)
	if err != nil {
		return nil, fmt.Errorf("json.Marshal tvMeta: %w", err)
	}

	externalIds := []domain.ExternalId{id}
	externalIds = append(externalIds, show.ExternalIDs...)

	images := []domain.Image{
		{
			ID:           uuid.New(),
			Role:         domain.ImageRolePoster,
			Provider:     id.Provider,
			ExternalPath: show.Poster,
		},
		{
			ID:           uuid.New(),
			Role:         domain.ImageRoleBackdrop,
			Provider:     id.Provider,
			ExternalPath: show.Backdrop,
		},
	}

	now := time.Now()
	media := domain.Media{
		ID:                mediaID,
		Type:              MediaType,
		Title:             show.Title,
		Status:            show.Status,
		Monitored:         false,
		PrimaryExternalId: id,
		ExternalIds:       externalIds,
		Metadata:          tvMetaJSON,
		CreatedAt:         now,
		LastSync:          now,
		UpdatedAt:         now,
		Images:            images,
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

// shorten is safe
func shorten(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
