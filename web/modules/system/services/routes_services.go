package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/bketelsen/omnius/web/stores"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar/sdk/go"
)

func SetupServicesRoutes(r chi.Router, cli *client.Client, ns *embeddednats.Server, stores *stores.KVStores, ctx context.Context) error {
	r.Route("/services", func(serviceRouter chi.Router) {

		// End proof of concept
		serviceRouter.Route("/", func(servicesRouter chi.Router) {

			servicesRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
				systemdConnection, _ := dbus.NewSystemConnectionContext(context.Background())
				defer systemdConnection.Close()
				var (
					units []dbus.UnitStatus
					err   error
				)
				if units, err = systemdConnection.ListUnitsByPatternsContext(context.Background(), []string{"running"}, []string{"*.service"}); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				ServicesPage(units).Render(r.Context(), w)
			})

		})
		serviceRouter.Get("/poll", func(w http.ResponseWriter, r *http.Request) {
			sse := datastar.NewSSE(w, r)

			// Watch for system updates
			ctx := r.Context()
			syswatcher, err := stores.SystemStore.WatchAll(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer syswatcher.Stop()

			for {
				select {
				case <-ctx.Done():
					return

				case entry := <-syswatcher.Updates():
					//		slog.Info("System Update", "entry", entry)
					if entry == nil {
						continue
					}
					switch k := entry.Key(); k {

					case "services":
						//	slog.Info("CPU Update")
						var v []dbus.UnitStatus
						if err := json.Unmarshal(entry.Value(), &v); err != nil {
							slog.Error("Service Update", "error", err)
							sse.ConsoleError(err)
							continue
						}
						c := ServiceOverviewCard(v)
						if err := sse.MergeFragmentTempl(c); err != nil {
							sse.ConsoleError(err)
							return
						}

					}
				}
			}
		})

	})
	return nil
}
