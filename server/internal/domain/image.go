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

// Image is a stored image reference with an assigned ID and source.
type Image struct {
	ID           uuid.UUID
	Role         ImageRole
	Source       ProviderName
	ExternalPath string
}

type ImageURL string

// ImageResolver turns a stored Image reference into a usable URL.
type ImageResolver interface {
	Resolve(path string, quality ImageQuality, role ImageRole) ImageURL
}

// ProviderImage is an image reference returned by a provider.
// It carries only the role and the provider-specific path; the handler
// assigns a UUID and source when converting to Image.
type ProviderImage struct {
	Role ImageRole
	Path string
}
