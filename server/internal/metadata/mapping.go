package metadata

import (
	"context"
	"server/internal/core"
)

type MappingData struct {
	Version string
	Entries []MappingEntry
}

type MappingEntry struct {
	IDs     []core.ExternalId
	Seasons []SeasonMapping
}

type SeasonMapping struct {
	Provider     string
	SeasonNumber int
}

type IDRow struct {
	GroupID    int
	Provider   string
	ProviderID string
}

type SeasonRow struct {
	GroupID        int
	Provider       string
	ProviderID     string
	TargetProvider string
	SeasonNumber   int
}

type MappingSource interface {
	Name() string
	Load(ctx context.Context, lastVersion string) (*MappingData, error)
}

type MappingRepository interface {
	ReplaceMappings(ctx context.Context, source string, ids []IDRow, seasons []SeasonRow) error
	GetSourceVersion(ctx context.Context, source string) (string, error)
	SetSourceVersion(ctx context.Context, source string, version string) error
	FindMappings(ctx context.Context, id core.ExternalId) ([]core.ExternalId, error)
}
