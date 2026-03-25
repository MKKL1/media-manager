package metadata

import "server/internal/domain"

type MappingData struct {
	Version string
	Entries []MappingEntry
}

type MappingEntry struct {
	IDs     []domain.ExternalId
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
