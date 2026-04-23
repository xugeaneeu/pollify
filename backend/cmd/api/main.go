package main

import (
	"context"
	"log"

	"xugeaneeu/pollify/internal/platform/app"
	"xugeaneeu/pollify/internal/platform/config"
	"xugeaneeu/pollify/internal/platform/logging"
)

func main() {
	if err := app.Run(context.Background(), config.Load(), logging.New()); err != nil {
		log.Fatal(err)
	}
}
