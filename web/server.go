package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bketelsen/omnius/web/modules/containers/docker"
	"github.com/bketelsen/omnius/web/modules/containers/incus"
	"github.com/bketelsen/omnius/web/modules/system"
	"github.com/bketelsen/omnius/web/modules/system/logs"
	"github.com/bketelsen/omnius/web/modules/system/networking"
	"github.com/bketelsen/omnius/web/modules/system/storage"
	"github.com/delaneyj/toolbelt"
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

		if err := errors.Join(
			setupHomeRoutes(router),
			system.SetupSystemRoutes(router, client),
			logs.SetupLogsRoutes(router, client),
			storage.SetupStorageRoutes(router, client),
			networking.SetupNetworkingRoutes(router, client),

			docker.SetupDockerRoutes(router, client),
			incus.SetupIncusRoutes(router, client),
		); err != nil {
			return fmt.Errorf("error setting up routes: %w", err)
		}

		srv := &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
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
