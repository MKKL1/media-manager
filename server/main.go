package main

import (
	"context"
	"fmt"
	stdHttp "net/http"
	"os"
	"os/signal"
	"server/infra/database"
	"server/infra/riverqueue"
	"server/internal/handler/http"
	"server/internal/media/movie"
	"server/internal/media/tv"
	"server/internal/metadata"
	"server/internal/metadata/services"
	"server/plugins/tmdb"

	"github.com/riverqueue/river"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				fmt.Println("shutting down...")
				return
			}
		}
	}()

	db, err := database.NewDB(os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	if err := database.Migrate(ctx, db); err != nil {
		panic(err)
	}

	repo := database.NewBunMediaRepository(db)
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	ctx = logger.WithContext(ctx)
	router := http.NewRouter(logger)

	provider := tmdb.NewProvider(os.Getenv("TMDB_API_KEY"))

	movieFetchers := map[string]movie.Fetcher{"tmdb": provider}
	tvFetchers := map[string]tv.Fetcher{"tmdb": provider}

	movieHandler := movie.NewMovieHandler(movieFetchers)
	tvHandler := tv.NewTVHandler(tvFetchers)

	handlers := metadata.Handlers{
		movie.MediaType: movieHandler,
		tv.MediaType:    tvHandler,
	}

	workers := river.NewWorkers()
	riverclient, err := riverqueue.NewClient(ctx, os.Getenv("DATABASE_URL"), workers)
	if err != nil {
		panic(err)
	}
	taskQueue := riverqueue.NewTaskQueue(riverclient)
	mediaPullService := services.NewPullService(repo, taskQueue, handlers)
	river.AddWorker(workers, riverqueue.NewPullMediaWorker(mediaPullService, logger))

	if err := riverclient.Start(ctx); err != nil {
		panic(err)
	}

	mediaController := http.NewMediaController(mediaPullService, logger)
	mediaController.Route(router)

	//res, err := searchService.SearchWithProvider(ctx, "one piece", core.MediaTypeTV, provider)
	//if err != nil {
	//	log.Fatal().Err(err).Msg("")
	//}
	//
	//_, err = riverclient.Insert(ctx, riverqueue.MediaPullArgs{
	//	ExtID:     res[0].ExternalID,
	//	MediaType: res[0].MediaType,
	//}, nil)
	//if err != nil {
	//	log.Fatal().Err(err).Msg("")
	//}

	err = stdHttp.ListenAndServe(":3000", router)
	if err != nil {
		panic(err)
	}

	<-ctx.Done()

	if err := riverclient.Stop(ctx); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	fmt.Println("Program exited gracefully.")

}
