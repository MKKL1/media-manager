package metadata

import (
	"context"
	"server/internal/core"
)

const (
	ProviderTMDBTV           = "tmdb:tv"
	ProviderTMDBMovie        = "tmdb:movie"
	ProviderTMDBSeason       = "tmdb:season"
	ProviderTMDBEpisode      = "tmdb:episode"
	ProviderTMDBEpisodeGroup = "tmdb:eg"
	ProviderTVDB             = "tvdb"
	ProviderIMDB             = "imdb"
	ProviderAniDB            = "anidb"
	ProviderMAL              = "mal"
	ProviderAniList          = "anilist"
	ProviderWikidata         = "wikidata"
)

type ExternalIDSource interface {
	Name() string
	GetExternalIDs(ctx context.Context, id string) ([]core.ExternalId, error)
}
