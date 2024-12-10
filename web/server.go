package web

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/a-h/templ"
	"github.com/bketelsen/omnius/web/layouts"
	"github.com/bketelsen/omnius/web/modules"

	// register modules
	_ "github.com/bketelsen/omnius/web/modules/containers/docker"
	_ "github.com/bketelsen/omnius/web/modules/containers/incus"

	_ "github.com/bketelsen/omnius/web/modules/system"
	_ "github.com/bketelsen/omnius/web/modules/system/storage"

	"github.com/bketelsen/omnius/web/stores"

	"github.com/delaneyj/toolbelt"
	"github.com/delaneyj/toolbelt/embeddednats"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	Port       int
	Logger     *slog.Logger
	Categories []*layouts.SidebarGroup
}

func NewServer(port int, logger *slog.Logger) *Server {
	return &Server{
		Port:   port,
		Logger: logger,
		Categories: []*layouts.SidebarGroup{
			{
				ID:    "omnius",
				Label: "OMNIUS",
			},
		},
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

		// setup sidebar groups
		for k, v := range modules.AvailableModules {
			s.Logger.Info("Checking module", slog.String("module", k))
			found := false

			for _, c := range s.Categories {
				if c.ID == v.Group() {
					s.Logger.Info("found", "category", v.Group(), "enabled", v.Enabled())
					found = true
					c.Links = append(c.Links, &layouts.SidebarLink{
						ID:         k,
						URL:        templ.SafeURL(fmt.Sprintf("/%s", k)),
						Label:      strings.ToUpper(k),
						IsDisabled: false,
					})
					break
				}

			}
			if !found {
				s.Logger.Info("not found", "category", v.Group(), "enabled", v.Enabled())
				s.Categories = append(s.Categories, &layouts.SidebarGroup{
					ID:    v.Group(),
					Label: strings.ToUpper(v.Group()),
					Links: []*layouts.SidebarLink{
						{
							ID:         k,
							URL:        templ.SafeURL(fmt.Sprintf("/%s", k)),
							Label:      strings.ToUpper(k),
							IsDisabled: false,
						},
					},
				})
			}
		}
		fmt.Println(s.Categories)
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
				v.SetupRoutes(router, s.Categories, ctx)
				go v.Poll(ctx)
			}
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
