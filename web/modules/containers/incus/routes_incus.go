package incus

import (
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar/code/go/sdk"
)

func SetupIncusRoutes(r chi.Router, cli *client.Client) error {

	r.Route("/incus", func(incusRouter chi.Router) {

		incusRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var (
				containers []types.Container
				err        error
			)
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			IncusPage(containers).Render(r.Context(), w)
		})

		incusRouter.Get("/api", func(w http.ResponseWriter, r *http.Request) {
			var (
				containers []types.Container
				err        error
			)
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)
			// do it quick to avoid page delay
			c := incusContainer(containers)

			if err := sse.MergeFragmentTempl(c); err != nil {
				sse.ConsoleError(err)
				return
			}
			ctx := r.Context()
			// now loop
			for {
				select {
				case <-ctx.Done():
					return
				case <-time.After(1 * time.Second):
					if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					c := incusContainer(containers)

					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}
				}
			}

		})
		incusRouter.Post("/api/{id}/pause", func(w http.ResponseWriter, r *http.Request) {

			idParam := chi.URLParam(r, "id")
			var (
				containers []types.Container
				err        error
			)
			if err = cli.ContainerPause(r.Context(), idParam); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)

			sse.MergeFragmentTempl(incusContainer(containers))

		})
		incusRouter.Post("/api/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			var (
				containers []types.Container
				err        error
				timeout    int
			)
			timeout = 10
			if err = cli.ContainerStop(r.Context(), idParam, containertypes.StopOptions{Timeout: &timeout}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			time.Sleep(10 * time.Second)
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)
			sse.MergeFragmentTempl(incusContainer(containers))

		})
		incusRouter.Post("/api/{id}/unpause", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			var (
				containers []types.Container
				err        error
			)
			if err = cli.ContainerUnpause(r.Context(), idParam); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			sse := datastar.NewSSE(w, r)
			sse.MergeFragmentTempl(incusContainer(containers))

		})

	})
	return nil
}
