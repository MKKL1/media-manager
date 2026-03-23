package metadata

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
)

type MappingService struct {
	repo    MappingRepository
	sources map[string]MappingSource
}

func NewMappingService(repo MappingRepository, sources []MappingSource) *MappingService {
	m := make(map[string]MappingSource, len(sources))
	for _, s := range sources {
		m[s.Name()] = s
	}
	return &MappingService{repo: repo, sources: m}
}

func (s *MappingService) SyncAll(ctx context.Context) error {
	log := zerolog.Ctx(ctx)

	for _, src := range s.sources {
		if err := s.syncOne(ctx, src); err != nil {
			return fmt.Errorf("sync %s: %w", src.Name(), err)
		}
		log.Info().Str("source", src.Name()).Msg("mapping source synced")
	}
	return nil
}

func (s *MappingService) syncOne(ctx context.Context, src MappingSource) error {
	lastVersion, err := s.repo.GetSourceVersion(ctx, src.Name())
	if err != nil {
		return fmt.Errorf("get version: %w", err)
	}

	data, err := src.Load(ctx, lastVersion)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	if data == nil {
		zerolog.Ctx(ctx).Debug().Str("source", src.Name()).Msg("unchanged, skipping")
		return nil
	}

	ids, seasons := flattenEntries(data.Entries)

	if err := s.repo.ReplaceMappings(ctx, src.Name(), ids, seasons); err != nil {
		return fmt.Errorf("replace mappings: %w", err)
	}

	return s.repo.SetSourceVersion(ctx, src.Name(), data.Version)
}

func flattenEntries(entries []MappingEntry) ([]IDRow, []SeasonRow) {
	groupIndex := make(map[string]int)
	nextGroup := 1
	var ids []IDRow
	var seasons []SeasonRow
	seen := make(map[string]struct{})

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
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			ids = append(ids, IDRow{GroupID: groupID, Provider: id.Provider, ProviderID: id.Id})
		}
		src := e.IDs[0]
		for _, sm := range e.Seasons {
			seasons = append(seasons, SeasonRow{
				GroupID: groupID, Provider: src.Provider, ProviderID: src.Id,
				TargetProvider: sm.Provider, SeasonNumber: sm.SeasonNumber,
			})
		}
	}
	return ids, seasons
}
