package jobs

import "server/internal/core"

var _ core.JobArgs = (*MediaPullArgs)(nil)

type MediaPullArgs struct {
	ExtID     core.ExternalId `json:"media_ext_id"`
	MediaType core.MediaType  `json:"media_type"`
}

func (MediaPullArgs) Kind() string { return "pull_media" }
