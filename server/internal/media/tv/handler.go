package tv

import (
	"context"
	"fmt"
	"time"

	json "github.com/bytedance/sonic"
	"github.com/google/uuid"

	"server/internal/domain"
	"server/internal/metadata"
)

const summaryMaxLength = 150

var _ metadata.MediaHandler = (*Handler)(nil)

type Handler struct {
	fetchers       map[domain.ProviderName]Fetcher
	imageResolvers map[domain.ProviderName]domain.ImageResolver
}

func NewHandler(
	fetchers map[domain.ProviderName]Fetcher,
	imageResolvers map[domain.ProviderName]domain.ImageResolver,
) *Handler {
	return &Handler{fetchers: fetchers, imageResolvers: imageResolvers}
}

func (h *Handler) Type() domain.MediaType { return MediaType }

func (h *Handler) FetchMedia(ctx context.Context, id domain.MediaIdentity) (*domain.MediaWithItems, error) {
	fetcher, ok := h.fetchers[id.Kind.ProviderName]
	if !ok {
		return nil, fmt.Errorf("provider %s not found for tv: %w", id.Kind.ProviderName, domain.ErrNoProvider)
	}

	result, err := fetcher.FetchShow(ctx, id.ID)
	if err != nil {
		return nil, fmt.Errorf("fetch show %s: %w", id, err)
	}

	mediaID := domain.GenerateMediaID()
	now := time.Now()

	seasonsMeta, items, err := mapSeasonsAndItems(result.Seasons, mediaID)
	if err != nil {
		return nil, err
	}

	meta := Metadata{
		OriginalTitle:    result.OriginalTitle,
		OriginalLanguage: result.OriginalLanguage,
		Overview:         result.Overview,
		Tagline:          result.Tagline,
		FirstAirDate:     result.FirstAirDate,
		LastAirDate:      result.LastAirDate,
		SeasonCount:      result.SeasonCount,
		EpisodeCount:     result.EpisodeCount,
		Runtime:          result.Runtime,
		Genres:           result.Genres,
		Tags:             result.Tags,
		Seasons:          seasonsMeta,
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal tv metadata: %w", err)
	}

	images := mapProviderImages(result.Images, id.Kind.ProviderName)
	externalIDs := append([]domain.MediaIdentity{id}, result.ExternalIDs...)

	status := ""
	if result.Status != nil {
		status = *result.Status
	}

	media := domain.Media{
		ID:                mediaID,
		Type:              MediaType,
		Title:             result.Title,
		Status:            status,
		Monitored:         false,
		PrimaryIdentity:   id,
		RelatedIdentities: externalIDs,
		Metadata:          metaJSON,
		Images:            images,
		CreatedAt:         now,
		LastSync:          now,
		UpdatedAt:         now,
	}

	return &domain.MediaWithItems{Media: media, Items: items}, nil
}

func (h *Handler) ToSummary(media domain.Media) (domain.MediaSummary, error) {
	var meta Metadata
	if len(media.Metadata) > 0 {
		if err := json.Unmarshal(media.Metadata, &meta); err != nil {
			return domain.MediaSummary{}, fmt.Errorf("unmarshal tv metadata: %w", err)
		}
	}

	posterPath, err := h.resolvePoster(media)
	if err != nil {
		return domain.MediaSummary{}, err
	}

	var firstAirDate time.Time
	if meta.FirstAirDate != nil {
		firstAirDate = *meta.FirstAirDate
	}

	return domain.MediaSummary{
		ID:              media.ID,
		Type:            media.Type,
		Title:           media.Title,
		OriginalTitle:   ptrOr(meta.OriginalTitle, ""),
		OriginalLang:    ptrOr(meta.OriginalLanguage, ""),
		Monitored:       media.Monitored,
		Status:          media.Status,
		Summary:         truncate(ptrOr(meta.Overview, ""), summaryMaxLength),
		ReleaseDate:     firstAirDate,
		PrimaryIdentity: media.PrimaryIdentity,
		PosterPath:      posterPath,
	}, nil
}

func (h *Handler) resolvePoster(media domain.Media) (domain.ImageURL, error) {
	resolver, ok := h.imageResolvers[media.PrimaryIdentity.Kind.ProviderName]
	if !ok {
		return "", fmt.Errorf(
			"image resolver for %s not found: %w",
			media.PrimaryIdentity.Kind.ProviderName,
			domain.ErrNoProvider,
		)
	}
	for _, img := range media.Images {
		if img.Role == domain.ImageRolePoster {
			return resolver.Resolve(img.ExternalPath, domain.ImageQualityThumb, img.Role), nil
		}
	}
	return "", nil
}

func mapSeasonsAndItems(seasons []SeasonResult, mediaID domain.MediaID) ([]SeasonMetadata, []domain.MediaItem, error) {
	var (
		seasonsMeta []SeasonMetadata
		items       []domain.MediaItem
	)
	for _, s := range seasons {
		for _, ep := range s.Episodes {
			epMeta := EpisodeMetadata{
				OriginalTitle: &ep.Title,
				Overview:      ep.Overview,
				AirDate:       ep.AirDate,
				Runtime:       ep.Runtime,
				Still:         ep.Still,
				SeasonNumber:  ep.SeasonNumber,
				EpisodeNumber: ep.EpisodeNumber,
			}
			epJSON, err := json.Marshal(epMeta)
			if err != nil {
				return nil, nil, fmt.Errorf("marshal episode metadata: %w", err)
			}
			items = append(items, domain.MediaItem{
				ID:       uuid.New(),
				MediaId:  mediaID,
				Status:   "Unknown",
				Metadata: epJSON,
			})
		}
		seasonsMeta = append(seasonsMeta, SeasonMetadata{
			SeasonNumber:         s.Number,
			EpisodeCount:         len(s.Episodes),
			EpisodeReleasedCount: len(s.Episodes), // TODO derive from air dates
		})
	}
	return seasonsMeta, items, nil
}

func mapProviderImages(images []domain.ProviderImage, source domain.ProviderName) []domain.Image {
	out := make([]domain.Image, 0, len(images))
	for _, img := range images {
		out = append(out, domain.Image{
			ID:           uuid.New(),
			Role:         img.Role,
			Source:       source,
			ExternalPath: img.Path,
		})
	}
	return out
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

// ptrOr dereferences a pointer, returning fallback if nil.
func ptrOr[T any](p *T, fallback T) T {
	if p != nil {
		return *p
	}
	return fallback
}
