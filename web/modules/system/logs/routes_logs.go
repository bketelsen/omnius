package logs

import (
	"net/http"
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	datastar "github.com/starfederation/datastar/code/go/sdk"
)

func SetupLogsRoutes(r chi.Router, cli *client.Client) error {

	r.Route("/logs", func(logsRouter chi.Router) {

		logsRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			var (
				containers []types.Container
				err        error
			)
			if containers, err = cli.ContainerList(r.Context(), containertypes.ListOptions{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			LogsPage(containers).Render(r.Context(), w)
		})

		logsRouter.Get("/api", func(w http.ResponseWriter, r *http.Request) {
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
			c := dockerContainer(containers)

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
					c := dockerContainer(containers)

					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}
				}
			}

		})
		logsRouter.Post("/api/{id}/pause", func(w http.ResponseWriter, r *http.Request) {

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

			sse.MergeFragmentTempl(dockerContainer(containers))

		})
		logsRouter.Post("/api/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
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
			sse.MergeFragmentTempl(dockerContainer(containers))

		})
		logsRouter.Post("/api/{id}/unpause", func(w http.ResponseWriter, r *http.Request) {
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
			sse.MergeFragmentTempl(dockerContainer(containers))

		})

	})
	return nil
}
