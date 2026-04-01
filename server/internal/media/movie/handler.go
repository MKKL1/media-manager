package movie

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
		return nil, fmt.Errorf("provider %s not found for movie: %w", id.Kind.ProviderName, domain.ErrNoProvider)
	}

	result, err := fetcher.FetchMovie(ctx, id.ID)
	if err != nil {
		return nil, fmt.Errorf("fetch movie %s: %w", id, err)
	}

	mediaID := domain.GenerateMediaID()
	now := time.Now()

	meta := Metadata{
		OriginalTitle:    result.OriginalTitle,
		OriginalLanguage: result.OriginalLanguage,
		Overview:         result.Overview,
		Tagline:          result.Tagline,
		ReleaseDate:      result.ReleaseDate,
		Runtime:          result.Runtime,
		Genres:           result.Genres,
		Tags:             result.Tags,
	}

	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal movie metadata: %w", err)
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

func (h *Handler) ToSummary(media domain.Media) (domain.MediaSummary, error) {
	var meta Metadata
	if len(media.Metadata) > 0 {
		if err := json.Unmarshal(media.Metadata, &meta); err != nil {
			return domain.MediaSummary{}, fmt.Errorf("unmarshal movie metadata: %w", err)
		}
	}

	posterPath, err := h.resolvePoster(media)
	if err != nil {
		return domain.MediaSummary{}, err
	}

	var releaseDate time.Time
	if meta.ReleaseDate != nil {
		releaseDate = *meta.ReleaseDate
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
		ReleaseDate:     releaseDate,
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

func ptrOr[T any](p *T, fallback T) T {
	if p != nil {
		return *p
	}
	return fallback
}
