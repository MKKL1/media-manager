package domain

import (
	"time"

	"github.com/google/uuid"
)

// Define models for workflow visualization
type WorkflowOverview struct {
	Id        uuid.UUID
	Name      string
	Type      string
	Issuer    WorkflowIssuer
	Status    string
	StartTime time.Time
	EndTime   time.Time
}

const (
	UserIssuerType   = "user"
	SystemIssuerType = "system"
)

type WorkflowIssuer struct {
	Type string
	Id   uuid.UUID
}

type ActivityOverview struct {
	Id        uuid.UUID
	Name      string
	Type      string
	Status    string
	StartTime time.Time
	EndTime   time.Time
}
