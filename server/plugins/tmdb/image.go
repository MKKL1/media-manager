package tmdb

import "server/internal/domain"

// tmdb has static image links so in this case this call is very cheap
// if other services generate temporary url, this abstraction helps
// TODO it may be necessary to set expiration time here, maybe

const imageBaseURL = "https://image.tmdb.org/t/p/"

// now it's hard coded here, but tmdb exposes https://api.themoviedb.org/3/configuration
// it may not be necessary to use it
var qualityMap = map[domain.ImageQuality]map[domain.ImageRole]string{
	domain.ImageQualityThumb: {
		domain.ImageRolePoster:   "w185",
		domain.ImageRoleBackdrop: "w300",
		domain.ImageRoleStill:    "w185",
	},
	domain.ImageQualityMedium: {
		domain.ImageRolePoster:   "w342",
		domain.ImageRoleBackdrop: "w780",
		domain.ImageRoleStill:    "w300",
	},
	domain.ImageQualityOriginal: {
		domain.ImageRolePoster:   "original",
		domain.ImageRoleBackdrop: "original",
		domain.ImageRoleStill:    "original",
	},
}

func (p *Provider) Resolve(img domain.Image, quality domain.ImageQuality) string {
	sizes, ok := qualityMap[quality]
	if !ok {
		sizes = qualityMap[domain.ImageQualityMedium]
	}
	size, ok := sizes[img.Role]
	if !ok {
		size = "original"
	}
	return imageBaseURL + size + img.ExternalPath
}
