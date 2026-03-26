package domain

import "github.com/google/uuid"

type ImageRole string

const (
	ImageRolePoster   ImageRole = "poster"
	ImageRoleBackdrop ImageRole = "backdrop"
	ImageRoleStill    ImageRole = "still"
	ImageRoleCover    ImageRole = "cover"
)

type ImageQuality string

const (
	ImageQualityThumb    ImageQuality = "thumb"
	ImageQualityMedium   ImageQuality = "medium"
	ImageQualityOriginal ImageQuality = "original"
)

type Image struct {
	ID           uuid.UUID
	Role         ImageRole
	Provider     string
	ExternalPath string
}

// ImageResolver turns a stored Image reference into a usable URL.
type ImageResolver interface {
	Resolve(img Image, quality ImageQuality) string
}
