package metadata

import (
	"context"
	"server/internal/domain"
	"time"
)

// MappingRepository stores and queries provider ID mappings.
type MappingRepository interface {
	ReplaceMappings(ctx context.Context, source string, ids []IDRow, seasons []SeasonRow) error
	GetSourceVersion(ctx context.Context, source string) (string, time.Time, error)
	SetSourceVersion(ctx context.Context, source string, version string) error
	FindMappings(ctx context.Context, id domain.MediaIdentity) ([]domain.MediaIdentity, error)
	MarkAttempt(ctx context.Context, source string) error
}
