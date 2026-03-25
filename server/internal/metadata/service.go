package metadata

import (
	"context"
	"fmt"
	"server/internal/domain"
)

type Service struct {
	repo     MediaRepository
	handlers Handlers
}

func NewService(repo MediaRepository, handlers Handlers) *Service {
	return &Service{repo: repo, handlers: handlers}
}

func (s *Service) List(ctx context.Context, q domain.MediaQuery) (domain.MediaPage, error) {
	list, total, err := s.repo.List(ctx, q)
	if err != nil {
		return domain.MediaPage{}, fmt.Errorf("s.repo.List: %w", err)
	}

	//get TV or movie specific handler
	handler, err := s.handlers.Get(q.Type)
	if err != nil {
		return domain.MediaPage{}, fmt.Errorf("s.handlers.Get: %w", err)
	}

	summaries := make([]domain.MediaSummary, 0, len(list))
	for _, m := range list {
		summary, err := handler.ToSummary(m)
		if err != nil {
			return domain.MediaPage{}, fmt.Errorf("handler.ToSummary: %w", err)
		}
		summaries = append(summaries, summary)
	}

	return domain.MediaPage{
		Items:  summaries,
		Total:  total,
		Offset: q.Paginate.Offset,
		Limit:  q.Paginate.Limit,
	}, nil
}
