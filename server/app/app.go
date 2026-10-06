package app

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"server/internal/event"
	cleanhttp "server/internal/interface/http"
	"server/internal/library"
	librarypg "server/internal/library/postgres"
	"server/internal/plugin"
	"server/internal/plugin/wasm"
	"server/internal/user"
	userpg "server/internal/user/postgres"
	"server/internal/view"
	anime_list "server/plugins/anime-list"
	"server/plugins/tvdb"
)

const adminGroup = "admins"

type App struct {
	db         *pgxpool.Pool
	httpServer *http.Server
}

type ShutdownFunc func(ctx context.Context) error

func New(ctx context.Context, cfg *Config) (*App, ShutdownFunc, error) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	if cfg.Log.Level == "debug" {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		logger.Info().Msgf("Log level set to debug")
	}

	db, err := openDB(ctx, cfg.Database.DSN())
	if err != nil {
		return nil, nil, err
	}

	for _, migrate := range []func(context.Context, *pgxpool.Pool) error{librarypg.Migrate, userpg.Migrate} {
		if err := migrate(ctx, db); err != nil {
			db.Close()
			return nil, nil, err
		}
	}

	events := &event.Bus{}
	lib, err := library.Open(ctx, librarypg.New(db), events)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("open library: %w", err)
	}

	users := user.NewService(userpg.New(db), events)
	authn := user.Authenticators{
		user.StaticToken{Token: cfg.Auth.AdminToken, User: user.User{Name: "admin", Groups: []string{adminGroup}}},
		users,
	}
	authz := user.Authorizers{user.Privileged(adminGroup), users}
	users.AllAuthorizers = authz

	router := cleanhttp.NewRouter(logger, authn, authz)
	(&cleanhttp.UserController{Users: users}).RegisterRoutes(router)
	(&cleanhttp.EventController{Events: events, Authorizer: authz}).RegisterRoutes(router)
	(&cleanhttp.LibraryController{Library: lib}).RegisterRoutes(router)
	var builtIn []plugin.Plugin
	if cfg.TVDB.APIKey != "" {
		builtIn = append(builtIn, tvdb.NewPlugin(cfg.TVDB.APIKey, cfg.TVDB.PIN))
	}
	plugins, err := loadPlugins(ctx, cfg.Plugins, builtIn, events)
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	(&cleanhttp.PluginController{Plugins: plugins}).RegisterRoutes(router)
	(&cleanhttp.ViewController{Views: &view.Views{Library: lib, Types: plugins.Types}}).RegisterRoutes(router)
	(&cleanhttp.ProviderController{
		Library:  lib,
		Plugins:  plugins.Providers,
		Mappings: []plugin.Mappings{anime_list.NewPlugin("")},
	}).RegisterRoutes(router)

	srv := &http.Server{Addr: cfg.HTTP.Addr, Handler: router}

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := srv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http shutdown: %w", err))
		}
		db.Close()
		return errors.Join(errs...)
	}

	return &App{
		db:         db,
		httpServer: srv,
	}, shutdown, nil
}

type plugins struct {
	configs map[string]PluginConfig
	builtIn []plugin.Plugin
	events  *event.Bus

	mu        sync.RWMutex
	loaded    map[string]wasm.Loaded
	providers []plugin.Plugin
	types     []view.Type
}

func loadPlugins(ctx context.Context, configs map[string]PluginConfig, builtIn []plugin.Plugin, events *event.Bus) (*plugins, error) {
	p := &plugins{configs: configs, builtIn: builtIn, events: events, loaded: map[string]wasm.Loaded{}}
	for _, name := range p.Names() {
		if err := p.load(ctx, name); err != nil {
			return nil, err
		}
	}
	p.providers, p.types, _ = p.combine(p.loaded)
	return p, nil
}

func (p *plugins) Names() []string { return slices.Sorted(maps.Keys(p.configs)) }

func (p *plugins) Providers() []plugin.Plugin {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.providers
}

func (p *plugins) Types() []view.Type {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.types
}

func (p *plugins) Reload(ctx context.Context, name string) error {
	if err := p.load(ctx, name); err != nil {
		return fmt.Errorf("%w: %w", plugin.ErrNotLoaded, err)
	}
	p.events.Publish(ctx, event.Event{Type: "plugin.updated", Subject: "plugins/" + name})
	return nil
}

func (p *plugins) load(ctx context.Context, name string) error {
	c := p.configs[name]
	loaded, err := wasm.Load(ctx, wasm.Config{Path: c.Path, AllowedHosts: c.AllowedHosts, Config: c.Config})
	if err != nil {
		return fmt.Errorf("plugins.%s: %w", name, err)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	all := maps.Clone(p.loaded)
	all[name] = loaded
	providers, types, err := p.combine(all)
	if err != nil {
		return fmt.Errorf("plugins.%s: %w", name, err)
	}
	p.loaded, p.providers, p.types = all, providers, types
	log := zerolog.Ctx(ctx).Info().Str("path", c.Path)
	if l := loaded.Provider; l != nil {
		log = log.Str("provider", string(l.Provider())).Interface("tags", plugin.TagsOf(l))
	}
	if t := loaded.Type; t != nil {
		log = log.Interface("formats", t.Formats())
	}
	log.Msg("plugin loaded")
	return nil
}

func (p *plugins) combine(loaded map[string]wasm.Loaded) ([]plugin.Plugin, []view.Type, error) {
	var providers []plugin.Plugin
	var types []view.Type
	for _, name := range slices.Sorted(maps.Keys(loaded)) {
		if l := loaded[name].Provider; l != nil {
			if slices.ContainsFunc(providers, func(q plugin.Plugin) bool { return q.Provider() == l.Provider() }) {
				return nil, nil, fmt.Errorf("another plugin is already %s", l.Provider())
			}
			providers = append(providers, l)
		}
		if t := loaded[name].Type; t != nil {
			for _, format := range t.Formats() {
				if slices.ContainsFunc(types, func(u view.Type) bool { return slices.Contains(u.Formats(), format) }) {
					return nil, nil, fmt.Errorf("another plugin already formats %s", format)
				}
			}
			types = append(types, t)
		}
	}
	for _, b := range p.builtIn {
		if !slices.ContainsFunc(providers, func(q plugin.Plugin) bool { return q.Provider() == b.Provider() }) {
			providers = append(providers, b)
		}
	}
	return providers, types, nil
}

func (a *App) Start(ctx context.Context) error {
	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zerolog.Ctx(ctx).Fatal().Err(err).Msg("http server stopped")
		}
	}()

	return nil
}

func openDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return db, nil
}
