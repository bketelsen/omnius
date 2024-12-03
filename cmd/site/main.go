package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/bketelsen/omnius/web"
	"github.com/docker/docker/client"
)

const port = 4321

func main() {
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("error creating docker connection: %w", err)
	}
	defer cli.Close()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	return web.RunBlocking(logger, cli, port)(ctx)

}
