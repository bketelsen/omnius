package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/bketelsen/omnius/web"
)

const port = 4321

func main() {
	ctx := context.Background()
	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	server := web.NewServer(port, logger)

	return server.RunBlocking()(ctx)

}
