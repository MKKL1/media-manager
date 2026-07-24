package tv

import (
	"server/pkg"
	"server/pkg/domain"
)

type Module struct {
	pkg.Module
}

func NewModule(
	fetchers map[domain.ProviderName]Fetcher,
	resolvers map[domain.ProviderName]domain.ImageResolver) *Module {
	m := &Module{
		Module: pkg.Module{
			Type:    MediaType,
			Handler: NewHandler(fetchers, resolvers),
			Mount:   nil,
			Migrate: nil,
		},
	}

	return m
}
