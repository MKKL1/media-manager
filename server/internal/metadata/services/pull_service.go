package services

import (
	"context"
	"fmt"
	"server/internal/core"
	"server/internal/metadata/workflows"

	"github.com/cschleiden/go-workflows/client"
	"github.com/google/uuid"
)

type PullService struct {
	wfClient *client.Client
}

func NewPullService(wfClient *client.Client) *PullService {
	return &PullService{wfClient: wfClient}
}

func (s *PullService) RequestPull(ctx context.Context, extID core.ExternalId, mediaType core.MediaType) (string, error) {
	instanceID := uuid.New().String()

	_, err := s.wfClient.CreateWorkflowInstance(
		ctx,
		client.WorkflowInstanceOptions{InstanceID: instanceID},
		workflows.MediaPullWorkflow,
		workflows.MediaPullInput{ExtID: extID, MediaType: mediaType},
	)
	if err != nil {
		return "", fmt.Errorf("create pull workflow: %w", err)
	}

	return instanceID, nil
}
