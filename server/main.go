package main

import (
	"context"
	"os"
	"os/signal"
	"server/app"

	"github.com/rs/zerolog/log"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	conf, err := app.Load("config.yaml")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config.yaml")
	}

	application, shutdown, err := app.New(ctx, conf)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}
	defer shutdown(ctx)

	if err := application.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("")
	}

	<-ctx.Done()
}
