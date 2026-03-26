package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"server/infra/database"
	"server/infra/telemetry"
	wfinfra "server/infra/workflow"
	"server/internal/domain"
	apphttp "server/internal/http"
	"server/internal/media/movie"
	"server/internal/media/tv"
	"server/internal/metadata"
	anime_list "server/plugins/anime-list"
	"server/plugins/tmdb"
	"time"

	"github.com/cschleiden/go-workflows/client"
	"github.com/cschleiden/go-workflows/diag"
	"github.com/cschleiden/go-workflows/worker"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type App struct {
	db         *bun.DB
	wfWorker   *worker.Worker
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

	tracerShutdown, err := telemetry.Init(ctx, telemetry.Config{
		Enabled:     cfg.Telemetry.Enabled,
		Endpoint:    cfg.Telemetry.Endpoint,
		ServiceName: cfg.Telemetry.ServiceName,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("init tracer: %w", err)
	}

	db, err := database.NewDB(cfg.Database.DSN())
	if err != nil {
		tracerShutdown()
		return nil, nil, fmt.Errorf("open db: %w", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		return nil, nil, fmt.Errorf("migrate db: %w", err)
	}

	mediaRepo := database.NewBunMediaRepository(db)
	mappingsRepo := database.NewMappingRepository(db)

	mappingSvc := metadata.NewMappingService(mappingsRepo, []metadata.MappingSource{
		anime_list.NewSource(""),
	})
	if err := mappingSvc.SyncAll(ctx); err != nil {
		return nil, nil, fmt.Errorf("sync mappings: %w", err)
	}

	provider := tmdb.NewProvider(cfg.TMDB.APIKey)
	handlers := metadata.Handlers{
		movie.MediaType: movie.NewMovieHandler(map[string]movie.Fetcher{"tmdb": provider}),
		tv.MediaType:    tv.NewTVHandler(map[string]tv.Fetcher{"tmdb": provider}, map[string]domain.ImageResolver{"tmdb": provider}),
	}

	wfBackend := wfinfra.NewBackend(
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Database,
		logger,
	)
	wfWorker := worker.New(wfBackend, nil)
	wfClient := client.New(wfBackend)

	pullSvc := metadata.NewPullService(mediaRepo, handlers, wfClient, logger)
	pullSvc.Register(wfWorker)

	mdSvc := metadata.NewService(mediaRepo, handlers)
	router := apphttp.NewRouter(logger)
	apphttp.NewMediaController(pullSvc, mdSvc).Route(router)
	router.Handle("/diag/*", http.StripPrefix("/diag", diag.NewServeMux(wfBackend)))

	srv := &http.Server{Addr: cfg.HTTP.Addr, Handler: router}

	//err = chi.Walk(router, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
	//	fmt.Printf("[%s]: '%s' has %d middlewares\n", method, route, len(middlewares))
	//	return nil
	//})
	//if err != nil {
	//	return nil, nil, err
	//}

	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := srv.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("http shutdown: %w", err))
		}
		if err := wfWorker.WaitForCompletion(); err != nil {
			errs = append(errs, fmt.Errorf("worker shutdown: %w", err))
		}
		if err := db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("db close: %w", err))
		}
		tracerShutdown()
		return errors.Join(errs...)
	}

	return &App{
		db:         db,
		wfWorker:   wfWorker,
		httpServer: srv,
	}, shutdown, nil
}

func (a *App) Start(ctx context.Context) error {
	if err := a.wfWorker.Start(ctx); err != nil {
		return fmt.Errorf("start workflow worker: %w", err)
	}

	go func() {
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zerolog.Ctx(ctx).Fatal().Err(err).Msg("http server stopped")
		}
	}()

	return nil
}
