package services

import (
	"context"
	"fmt"
	"net/http"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
)

func SetupServicesRoutes(r chi.Router, cli *client.Client, ns *embeddednats.Server) error {
	// Proof of concept for systemd integration
	systemdConnection, _ := dbus.NewSystemConnectionContext(context.Background())

	listOfUnits, _ := systemdConnection.ListUnitsByPatternsContext(context.Background(), []string{"running"}, []string{"*.service"})

	for _, unit := range listOfUnits {
		fmt.Println(unit.Name)
	}

	systemdConnection.Close()
	// End proof of concept
	r.Route("/services", func(servicesRouter chi.Router) {

		servicesRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var (
				containers []types.Container
				err        error
			)
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			ServicesPage(containers).Render(r.Context(), w)
		})

	})
	return nil
}
