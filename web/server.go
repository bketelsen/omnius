package web

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/bketelsen/omnius/web/modules"
	// register modules
	_ "github.com/bketelsen/omnius/web/modules/containers/docker"
	_ "github.com/bketelsen/omnius/web/modules/system"

	"github.com/bketelsen/omnius/web/stores"

	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Port   int
	Logger *slog.Logger
}

func NewServer(port int, logger *slog.Logger) *Server {
	return &Server{
		Port:   port,
		Logger: logger,
	}
}

func (s *Server) RunBlocking() toolbelt.CtxErrFunc {
	s.Logger.Info(fmt.Sprintf("Starting Server @:%d", s.Port))

	return func(ctx context.Context) (err error) {

		router := chi.NewRouter()
		router.Use(middleware.Recoverer)
		router.Handle("/static/*", http.StripPrefix("/static/", static(s.Logger)))

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

		ctx, cancel := context.WithCancel(ctx)
		for k, v := range modules.AvailableModules {
			fmt.Println(k, v)
			s.Logger.Info("Creating module", slog.String("module", k))
			err := v.Init(s.Logger, kvstore, natsCon, js)
			if err != nil {
				s.Logger.Error("error creating module", slog.String("module", k), slog.String("error", err.Error()))
				continue
			}
			err = v.CreateStore(kvstore)
			if err != nil {
				s.Logger.Error("error creating store", slog.String("module", k), slog.String("error", err.Error()))
				continue
			}
			if v.Enabled() {
				v.SetupRoutes(router, ctx)
				go v.Poll(ctx)
			}
		}

		if err := errors.Join(
			setupHomeRoutes(router, ns),
			//	system.SetupSystemRoutes(router, ns, kvstore, ctx),
			//	services.SetupServicesRoutes(router, ns, kvstore, ctx),
			//	logs.SetupLogsRoutes(router, ns),
			//	storage.SetupStorageRoutes(router, ns),
			//	networking.SetupNetworkingRoutes(router, ns),
			//	dm.SetupRoutes(router, ctx),
			//  docker.SetupDockerRoutes(router, logger, client, ns, kvstore, ctx),
			//	incus.SetupIncusRoutes(router, ns),
		); err != nil {
			cancel()
			return fmt.Errorf("error setting up routes: %w", err)
		}

		srv := &http.Server{
			Addr:    fmt.Sprintf("0.0.0.0:%d", s.Port),
			Handler: router,
		}
		go func() {
			<-ctx.Done()
			cancel()
			defer s.Logger.Info("Stopping Server")

			srv.Shutdown(context.Background())
		}()

		return srv.ListenAndServe()
	}
}
