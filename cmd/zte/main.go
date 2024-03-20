package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"grono.dev/zte-5g/app"
)

func main() {
	setupLog()
	ctx := log.Logger.WithContext(context.TODO())
	ctx, _ = signal.NotifyContext(ctx, os.Interrupt)

	go func() {
		<-ctx.Done()
		<-time.After(1 * time.Second)
		log.Warn().Msg("app didn't terminate, force exit")
		os.Exit(2)
	}()

	err := app.Run(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Error().Err(err).Msg("app exited with error")
		os.Exit(1)
	}
	os.Exit(0)
}

func setupLog() {
	lvl := zerolog.InfoLevel
	switch os.Getenv("LOG") {
	case "":
	case "debug":
		lvl = zerolog.DebugLevel
	case "trace":
		lvl = zerolog.TraceLevel
	}

	zerolog.LevelColors[zerolog.DebugLevel] = 0

	// Green
	zerolog.LevelColors[zerolog.InfoLevel] = 42

	// Orange
	zerolog.LevelColors[zerolog.WarnLevel] = 43

	// Red
	zerolog.LevelColors[zerolog.ErrorLevel] = 41
	zerolog.LevelColors[zerolog.FatalLevel] = 41
	zerolog.LevelColors[zerolog.PanicLevel] = 41
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339Nano,
	}).Level(lvl)

}
