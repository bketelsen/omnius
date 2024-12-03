package system

import (
	"fmt"
	"net/http"
	"time"

	"github.com/docker/docker/client"
	"github.com/go-chi/chi/v5"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	datastar "github.com/starfederation/datastar/code/go/sdk"
)

func SetupSystemRoutes(r chi.Router, cli *client.Client) error {

	r.Route("/system", func(systemRouter chi.Router) {

		systemRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			v, err := mem.VirtualMemory()

			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			SystemPage(v).Render(r.Context(), w)
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
