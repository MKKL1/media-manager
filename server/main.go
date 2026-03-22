package main

import (
	"context"
	"fmt"
	stdHttp "net/http"
	"os"
	"os/signal"
	"server/infra/database"
	"server/infra/telemetry"
	wfinfra "server/infra/workflow"
	"server/internal/handler/http"
	"server/internal/media/movie"
	"server/internal/media/tv"
	"server/internal/metadata"
	"server/internal/metadata/services"
	"server/internal/metadata/workflows"
	anime_list "server/plugins/anime-list"
	"server/plugins/tmdb"

	"github.com/cschleiden/go-workflows/client"
	"github.com/cschleiden/go-workflows/worker"
	"github.com/rs/zerolog"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	shutdown, err := telemetry.InitTracer(ctx)
	if err != nil {
		panic(err)
	}
	defer shutdown()

	db, err := database.NewDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		panic(err)
	}

	repo := database.NewBunMediaRepository(db)

	mappingsRepo := database.NewMappingRepository(db)
	mappingsService := services.NewMappingsService(mappingsRepo, []metadata.MappingSource{
		anime_list.NewProvider(""),
	})
	if err := mappingsService.SyncMappings(ctx); err != nil {
		panic(err)
	}

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)

	provider := tmdb.NewProvider(os.Getenv("TMDB_API_KEY"))
	handlers := metadata.Handlers{
		movie.MediaType: movie.NewMovieHandler(map[string]movie.Fetcher{"tmdb": provider}),
		tv.MediaType:    tv.NewTVHandler(map[string]tv.Fetcher{"tmdb": provider}),
	}

	//TODO accept both url and params
	wfBackend := wfinfra.NewBackend("localhost", 5432, "user", "password", "my_project_db")
	wfWorker := worker.New(wfBackend, nil)
	workflows.RegisterPullWorkflow(wfWorker, repo, handlers)

	if err := wfWorker.Start(ctx); err != nil {
		panic(err)
	}

	wfClient := client.New(wfBackend)
	pullService := services.NewPullService(wfClient)

	router := http.NewRouter(logger)
	http.NewMediaController(pullService, logger).Route(router)

	go func() {
		<-ctx.Done()
		fmt.Println("shutting down...")
	}()

	if err := stdHttp.ListenAndServe(":3000", router); err != nil {
		logger.Fatal().Err(err).Msg("server stopped")
	}
}
