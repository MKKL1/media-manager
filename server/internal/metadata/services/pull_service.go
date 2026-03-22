package services

import (
	"context"
	"fmt"
	"server/internal/core"
	"server/internal/metadata/workflows"

	"github.com/cschleiden/go-workflows/client"
	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
)

type PullService struct {
	wfClient     *client.Client
	pullWorkflow func(workflow.Context, workflows.MediaPullInput) (workflows.MediaPullResult, error)
}

func NewPullService(
	wfClient *client.Client,
	pullWorkflow func(workflow.Context, workflows.MediaPullInput) (workflows.MediaPullResult, error),
) *PullService {
	return &PullService{wfClient: wfClient, pullWorkflow: pullWorkflow}
}

func (s *PullService) RequestPull(ctx context.Context, extID core.ExternalId, mediaType core.MediaType) (string, error) {
	instanceID := uuid.New().String()

	_, err := s.wfClient.CreateWorkflowInstance(
		ctx,
		client.WorkflowInstanceOptions{InstanceID: instanceID},
		s.pullWorkflow,
		workflows.MediaPullInput{ExtID: extID, MediaType: mediaType},
	)
	if err != nil {
		return "", fmt.Errorf("create pull workflow: %w", err)
	}

	return instanceID, nil
}
