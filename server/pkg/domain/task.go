package domain

import "context"

type JobArgs interface {
	Kind() string
}

type Job struct {
	Id        int64 `json:"id"`
	Duplicate bool  `json:"duplicate"`
}

type JobQueue interface {
	Enqueue(ctx context.Context, args JobArgs) (*Job, error)
}
