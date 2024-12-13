package docker

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bketelsen/omnius/web/layouts"
	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar/sdk/go"
)

func (dm *DockerModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up Docker Routes")

	r.Route("/docker", func(dockerRouter chi.Router) {

		dockerRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var (
				containers []types.Container
				images     []image.Summary
				err        error
			)
			if containers, err = dm.client.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			DockerPage(sidebarGroups, containers, images).Render(r.Context(), w)
		})

		dockerRouter.Get("/poll", func(w http.ResponseWriter, r *http.Request) {

			sse := datastar.NewSSE(w, r)
			ctwatcher, err := dm.Stores.DockerStore.Watch(ctx, "containers")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer ctwatcher.Stop()

			imgwatcher, err := dm.Stores.DockerStore.Watch(ctx, "images")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer imgwatcher.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-imgwatcher.Updates():
					//	slog.Info("Docker Update", "entry", entry)
					if entry == nil {
						continue
					}
					var cc []image.Summary
					if err := json.Unmarshal(entry.Value(), &cc); err != nil {
						dm.Logger.Error("Docker Update", "error", err)
						sse.ConsoleError(err)
						continue
					}
					c := ImageOverview(cc)
					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}
				case entry := <-ctwatcher.Updates():
					//	slog.Info("Docker Update", "entry", entry)
					if entry == nil {
						continue
					}
					var cc []types.Container
					if err := json.Unmarshal(entry.Value(), &cc); err != nil {
						dm.Logger.Error("Docker Update", "error", err)
						sse.ConsoleError(err)
						continue
					}
					c := DockerDetail(cc)
					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}

				}
			}

		})
		dockerRouter.Post("/api/{id}/pause", func(w http.ResponseWriter, r *http.Request) {

			idParam := chi.URLParam(r, "id")
			var (
				containers []types.Container
				err        error
			)
			if err = dm.client.ContainerPause(r.Context(), idParam); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if containers, err = dm.client.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)

			sse.MergeFragmentTempl(DockerContainer(containers))

		})
		dockerRouter.Post("/api/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			var (
				containers []types.Container
				err        error
				timeout    int
			)
			timeout = 10
			if err = dm.client.ContainerStop(r.Context(), idParam, containertypes.StopOptions{Timeout: &timeout}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			time.Sleep(10 * time.Second)
			if containers, err = dm.client.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)
			sse.MergeFragmentTempl(DockerContainer(containers))

		})
		dockerRouter.Post("/api/{id}/unpause", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			var (
				containers []types.Container
				err        error
			)
			if err = dm.client.ContainerUnpause(r.Context(), idParam); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if containers, err = dm.client.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)
			sse.MergeFragmentTempl(DockerContainer(containers))

		})

	})
	return nil
}
