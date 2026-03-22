package services

import (
	"context"
	"fmt"
	"server/internal/metadata"

	"github.com/rs/zerolog"
)

type MappingsService struct {
	repo      metadata.MappingRepository
	providers map[string]metadata.MappingSource
}

func NewMappingsService(repo metadata.MappingRepository, providers []metadata.MappingSource) *MappingsService {
	providerMap := make(map[string]metadata.MappingSource)
	for _, provider := range providers {
		providerMap[provider.Name()] = provider
	}

	return &MappingsService{
		repo:      repo,
		providers: providerMap,
	}
}

func (s MappingsService) SyncMappings(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	for _, pr := range s.providers {
		lastVersion, err := s.repo.GetSourceVersion(ctx, pr.Name())
		if err != nil {
			return fmt.Errorf("get version for %s: %w", pr.Name(), err)
		}

		data, err := pr.Load(ctx, lastVersion)
		if err != nil {
			return fmt.Errorf("load %s: %w", pr.Name(), err)
		}

		if data == nil {
			logger.Info().Str("source", pr.Name()).Msg("mapping source unchanged, skipping")
			continue
		}

		ids, seasons := groupEntries(data.Entries)

		if err := s.repo.ReplaceMappings(ctx, pr.Name(), ids, seasons); err != nil {
			return fmt.Errorf("repo replace mappings %s: %w", pr.Name(), err)
		}

		if err := s.repo.SetSourceVersion(ctx, pr.Name(), data.Version); err != nil {
			return fmt.Errorf("set version for %s: %w", pr.Name(), err)
		}

		logger.Info().
			Str("provider", pr.Name()).
			Str("version", data.Version).
			Int("entries", len(data.Entries)).
			Msg("mapping source synced")
	}
	return nil
}

func groupEntries(entries []metadata.MappingEntry) ([]metadata.IDRow, []metadata.SeasonRow) {
	groupIndex := make(map[string]int)
	nextGroup := 1

	var ids []metadata.IDRow
	var seasons []metadata.SeasonRow
	seenIDs := make(map[string]struct{})

	for _, e := range entries {
		if len(e.IDs) == 0 {
			continue
		}

		groupID := 0
		for _, id := range e.IDs {
			if gid, ok := groupIndex[id.String()]; ok {
				groupID = gid
				break
			}
		}
		if groupID == 0 {
			groupID = nextGroup
			nextGroup++
		}

		for _, id := range e.IDs {
			key := id.String()
			groupIndex[key] = groupID
			if _, exists := seenIDs[key]; exists {
				continue //skip duplicate
			}
			seenIDs[key] = struct{}{}
			ids = append(ids, metadata.IDRow{
				GroupID:    groupID,
				Provider:   id.Provider,
				ProviderID: id.Id,
			})
		}

		sourceID := e.IDs[0]
		for _, s := range e.Seasons {
			seasons = append(seasons, metadata.SeasonRow{
				GroupID:        groupID,
				Provider:       sourceID.Provider,
				ProviderID:     sourceID.Id,
				TargetProvider: s.Provider,
				SeasonNumber:   s.SeasonNumber,
			})
		}
	}

	return ids, seasons
}
