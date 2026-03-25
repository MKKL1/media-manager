package metadata

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"server/infra/telemetry"
	"server/internal/domain"

	"github.com/cschleiden/go-workflows/client"
	"github.com/cschleiden/go-workflows/worker"
	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PullInput struct {
	ExtID     domain.ExternalId `json:"ext_id"`
	MediaType domain.MediaType  `json:"media_type"`
}

type PullResult struct {
	MediaID string `json:"media_id"`
	Existed bool   `json:"existed"`
}

type PullService struct {
	repo     MediaRepository
	handlers Handlers
	wfClient *client.Client
	logger   zerolog.Logger // stored so activities can use it directly
}

func NewPullService(
	repo MediaRepository,
	handlers Handlers,
	wfClient *client.Client,
	logger zerolog.Logger,
) *PullService {
	return &PullService{
		repo:     repo,
		handlers: handlers,
		wfClient: wfClient,
		logger:   logger,
	}
}

func (s *PullService) Register(w *worker.Worker) {
	w.RegisterWorkflow(s.run)
	w.RegisterActivity(s.lookupExisting)
	w.RegisterActivity(s.fetchMetadata)
	w.RegisterActivity(s.storeMedia)
}

func (s *PullService) RequestPull(ctx context.Context, extID domain.ExternalId, mediaType domain.MediaType) (string, error) {
	instanceID := uuid.NewString()

	_, err := s.wfClient.CreateWorkflowInstance(
		ctx,
		client.WorkflowInstanceOptions{InstanceID: instanceID},
		s.run,
		PullInput{ExtID: extID, MediaType: mediaType},
	)
	if err != nil {
		return "", fmt.Errorf("create pull workflow: %w", err)
	}
	return instanceID, nil
}

func (s *PullService) run(
	ctx workflow.Context,
	input PullInput,
) (PullResult, error) {
	ctx, span := workflow.Tracer(ctx).Start(ctx, "media_pull", trace.WithAttributes(
		attribute.String("id", input.ExtID.String()),
	))
	defer span.End()

	logger := workflow.Logger(ctx)

	opts := workflow.ActivityOptions{
		RetryOptions: workflow.RetryOptions{MaxAttempts: 3},
	}

	existing, err := workflow.ExecuteActivity[*domain.Media](
		ctx, opts, s.lookupExisting, input.ExtID,
	).Get(ctx)
	if err != nil {
		return PullResult{}, fmt.Errorf("lookup %s: %w", input.ExtID, err)
	}
	if existing != nil {
		logger.Info("media already exists", slog.String("media_id", existing.ID.String()))
		return PullResult{MediaID: existing.ID.String(), Existed: true}, nil
	}

	media, err := workflow.ExecuteActivity[domain.MediaWithItems](
		ctx, opts, s.fetchMetadata, input.ExtID, input.MediaType,
	).Get(ctx)
	if err != nil {
		return PullResult{}, fmt.Errorf("fetch metadata %s: %w", input.ExtID, err)
	}

	if _, err := workflow.ExecuteActivity[struct{}](
		ctx, opts, s.storeMedia, media,
	).Get(ctx); err != nil {
		return PullResult{}, fmt.Errorf("store media %s: %w", media.Media.ID, err)
	}

	return PullResult{MediaID: media.Media.ID.String()}, nil
}

func (s *PullService) lookupExisting(
	ctx context.Context,
	extID domain.ExternalId,
) (_ *domain.Media, err error) {
	ctx, end := telemetry.Start(ctx, "metadata.LookupExisting",
		trace.WithAttributes(attribute.String("ext_id", extID.String())),
	)
	defer end(&err)

	media, err := s.repo.GetByExternalID(ctx, extID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup %s: %w", extID, err)
	}
	return media, nil
}

func (s *PullService) fetchMetadata(
	ctx context.Context,
	extID domain.ExternalId,
	mediaType domain.MediaType,
) (_ domain.MediaWithItems, err error) {
	ctx, end := telemetry.Start(ctx, "metadata.FetchMetadata",
		trace.WithAttributes(
			attribute.String("ext_id", extID.String()),
			attribute.String("media_type", string(mediaType)),
		),
	)
	defer end(&err)

	handler, err := s.handlers.Get(mediaType)
	if err != nil {
		return domain.MediaWithItems{}, err
	}

	m, err := handler.FetchMedia(ctx, extID)
	if err != nil {
		return domain.MediaWithItems{}, fmt.Errorf("fetch %s %s: %w", mediaType, extID, err)
	}

	s.logger.Info().
		Str("ext_id", extID.String()).
		Str("title", m.Media.Title).
		Msg("metadata fetched")

	return *m, nil
}

func (s *PullService) storeMedia(ctx context.Context, media domain.MediaWithItems) (_ struct{}, err error) {
	ctx, end := telemetry.Start(ctx, "metadata.StoreMedia",
		trace.WithAttributes(attribute.String("media_id", media.Media.ID.String())),
	)
	defer end(&err)

	if err := s.repo.StoreMediaWithItems(ctx, media); err != nil {
		return struct{}{}, fmt.Errorf("store media %s: %w", media.Media.ID, err)
	}
	return struct{}{}, nil
}
