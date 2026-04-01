package metadata

import (
	"context"
	"fmt"
	"server/internal/domain"
)

type Service struct {
	repo     domain.MediaRepository
	handlers Handlers
}

func NewService(repo domain.MediaRepository, handlers Handlers) *Service {
	return &Service{repo: repo, handlers: handlers}
}

func (s *Service) List(ctx context.Context, q domain.MediaQuery) (domain.MediaPage, error) {
	list, total, err := s.repo.List(ctx, q)
	if err != nil {
		return domain.MediaPage{}, fmt.Errorf("list media: %w", err)
	}

	summaries := make([]domain.MediaSummary, 0, len(list))
	for _, m := range list {
		handler, err := s.handlers.Get(m.Type)
		if err != nil {
			return domain.MediaPage{}, fmt.Errorf("get handler for %s: %w", m.Type, err)
		}
		summary, err := handler.ToSummary(m)
		if err != nil {
			return domain.MediaPage{}, fmt.Errorf("summarize %s: %w", m.ID, err)
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

func (s *Service) Get(ctx context.Context, id domain.MediaID) (domain.Media, error) {

}
