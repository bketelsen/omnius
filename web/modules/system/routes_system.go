package system

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/bketelsen/omnius/web/stores"
	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/shirou/gopsutil/cpu"
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
			SystemPage(v).Render(r.Context(), w)
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
			slog.Info("Start Polling System")

			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-syswatcher.Updates():
					slog.Info("System Update", "entry", entry)
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
		systemRouter.Get("/api/cpu", func(w http.ResponseWriter, r *http.Request) {
			cores, err := cpu.Counts(true)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			usage, err := cpu.Percent(0, false)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			used := fmt.Sprintf("%.2f", usage[0])

			// round usage to whole number
			usedPercent := fmt.Sprintf("%.0f", usage[0])
			sse := datastar.NewSSE(w, r)
			// do it quick to avoid page delay
			c := cpuDetailCard(cores, used, usedPercent)

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
					if cores, err = cpu.Counts(true); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					if usage, err = cpu.Percent(0, false); err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					used = fmt.Sprintf("%.2f", usage[0])
					usedPercent = fmt.Sprintf("%.0f", usage[0])

					c := cpuDetailCard(cores, used, usedPercent)

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
