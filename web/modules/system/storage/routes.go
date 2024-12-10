package storage

import (
	"context"
	"net/http"

	"github.com/bketelsen/omnius/web/layouts"
	"github.com/go-chi/chi/v5"
)

func (dm *StorageModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up Storage Routes")

	r.Route("/storage", func(storageRouter chi.Router) {

		storageRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {

			StoragePage(sidebarGroups).Render(r.Context(), w)
		})

	})
	return nil
}
