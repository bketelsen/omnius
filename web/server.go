package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bketelsen/omnius/web/stores"

	"github.com/bketelsen/omnius/web/modules/containers/docker"
	"github.com/bketelsen/omnius/web/modules/containers/incus"
	"github.com/bketelsen/omnius/web/modules/system"
	"github.com/bketelsen/omnius/web/modules/system/logs"
	"github.com/bketelsen/omnius/web/modules/system/networking"
	"github.com/bketelsen/omnius/web/modules/system/services"

	"github.com/bketelsen/omnius/web/modules/system/storage"
	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func RunBlocking(logger *slog.Logger, client *client.Client, port int) toolbelt.CtxErrFunc {
	logger.Info(fmt.Sprintf("Starting Server @:%d", port))
	return func(ctx context.Context) (err error) {

		router := chi.NewRouter()
		router.Use(middleware.Recoverer)
		router.Handle("/static/*", http.StripPrefix("/static/", static(logger)))
		natsPort, err := toolbelt.FreePort()
		if err != nil {
			return fmt.Errorf("error getting free port: %w", err)
		}

		ns, err := embeddednats.New(ctx, embeddednats.WithNATSServerOptions(&natsserver.Options{
			JetStream: true,
			Port:      natsPort,
		}))

		if err != nil {
			return fmt.Errorf("error creating embedded nats server: %w", err)
		}

		ns.WaitForServer()
		kvstore := &stores.KVStores{}
		natsCon, err := ns.Client()
		if err != nil {
			return fmt.Errorf("error getting nats client: %w", err)
		}
		js, err := jetstream.New(natsCon)
		if err != nil {
			return fmt.Errorf("error creating jetstream client: %w", err)
		}
		dm, err := docker.NewDockerModule(logger, kvstore, client, natsCon, js)
		if err != nil {
			return fmt.Errorf("error creating docker module: %w", err)
		}
		err = dm.CreateStore(kvstore)
		if err != nil {
			return fmt.Errorf("error creating jetstream client: %w", err)
		}
		if err := errors.Join(
			setupHomeRoutes(router, ns),
			system.SetupSystemRoutes(router, client, ns, kvstore, ctx),
			services.SetupServicesRoutes(router, client, ns, kvstore, ctx),
			logs.SetupLogsRoutes(router, client, ns),
			storage.SetupStorageRoutes(router, client, ns),
			networking.SetupNetworkingRoutes(router, client, ns),
			dm.SetupRoutes(router, ctx),
			//docker.SetupDockerRoutes(router, logger, client, ns, kvstore, ctx),
			incus.SetupIncusRoutes(router, client, ns),
		); err != nil {
			return fmt.Errorf("error setting up routes: %w", err)
		}

		go poll(ctx, ns, kvstore, dm)
		if err != nil {
			return fmt.Errorf("error polling: %w", err)
		}
		srv := &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", port),
			Handler: router,
		}
		go func() {
			<-ctx.Done()
			defer logger.Info("Stopping Server")

			srv.Shutdown(context.Background())
		}()

		return srv.ListenAndServe()
	}
}
