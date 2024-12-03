package system

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/bketelsen/omnius/web/modules/containers/docker"
	"github.com/bketelsen/omnius/web/stores"

	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/shirou/gopsutil/v4/mem"
	datastar "github.com/starfederation/datastar/code/go/sdk"
)

func SetupSystemRoutes(r chi.Router, cli *client.Client, ns *embeddednats.Server, stores *stores.KVStores, ctx context.Context) error {

	r.Route("/system", func(systemRouter chi.Router) {

		systemRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			v, err := mem.VirtualMemory()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			c := CPUSimple{
				UsedPercent: "0",
				Used:        "0",
				Cores:       0,
			}
			containers := []types.Container{}
			SystemPage(c, v, containers).Render(r.Context(), w)
		})

		systemRouter.Get("/poll", func(w http.ResponseWriter, r *http.Request) {
			sse := datastar.NewSSE(w, r)

			// Watch for system updates
			ctx := r.Context()
			syswatcher, err := stores.SystemStore.WatchAll(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer syswatcher.Stop()
			dockerwatcher, err := stores.DockerStore.Watch(ctx, "containers")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer dockerwatcher.Stop()
			slog.Info("Start Polling System")

			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-dockerwatcher.Updates():
					//	slog.Info("Docker Update", "entry", entry)
					if entry == nil {
						continue
					}
					var cc []types.Container
					if err := json.Unmarshal(entry.Value(), &cc); err != nil {
						slog.Error("Docker Update", "error", err)
						sse.ConsoleError(err)
						continue
					}
					c := docker.DockerContainer(cc)
					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}
				case entry := <-syswatcher.Updates():
					//		slog.Info("System Update", "entry", entry)
					if entry == nil {
						continue
					}
					switch k := entry.Key(); k {
					case "virtualMemory":
						slog.Info("Memory Update")
						var v mem.VirtualMemoryStat
						if err := json.Unmarshal(entry.Value(), &v); err != nil {
							slog.Error("Memory Update", "error", err)
							sse.ConsoleError(err)
							continue
						}
						c := memoryDetailCard(&v)
						if err := sse.MergeFragmentTempl(c); err != nil {
							sse.ConsoleError(err)
							return
						}

					case "cpu":
						slog.Info("CPU Update")
						var v CPUSimple
						if err := json.Unmarshal(entry.Value(), &v); err != nil {
							slog.Error("CPU Update", "error", err)
							sse.ConsoleError(err)
							continue
						}
						c := cpuDetailCard(v)
						if err := sse.MergeFragmentTempl(c); err != nil {
							sse.ConsoleError(err)
							return
						}
					}
				}
			}
		})
		systemRouter.Get("/api/memory", func(w http.ResponseWriter, r *http.Request) {
			v, err := mem.VirtualMemory()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			sse := datastar.NewSSE(w, r)
			// do it quick to avoid page delay
			c := memoryDetailCard(v)

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
					if v, err = mem.VirtualMemory(); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					c := memoryDetailCard(v)

					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}
				}
			}

		})

	})
	return nil
}

type CPUSimple struct {
	UsedPercent string `json:"usedPercent"`
	Used        string `json:"used"`
	Cores       int    `json:"cores"`
}
