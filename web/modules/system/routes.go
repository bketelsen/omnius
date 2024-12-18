package system

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/bketelsen/omnius/web/components"
	"github.com/bketelsen/omnius/web/layouts"
	"github.com/bketelsen/omnius/web/modules/containers/docker"
	"github.com/bketelsen/omnius/web/modules/system/services"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/docker/docker/api/types"
	"github.com/go-chi/chi/v5"
	"github.com/shirou/gopsutil/v4/mem"
	datastar "github.com/starfederation/datastar/sdk/go"
)

type CtxKey string

const (
	CtxKeyUser CtxKey = "user"
)

func UserFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(CtxKeyUser).(string)
	return userID, ok
}

func ContextWithUser(ctx context.Context, user string) context.Context {
	return context.WithValue(ctx, CtxKeyUser, user)
}

func (dm *SystemModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up System Routes")
	r.Route("/"+ModuleName, func(systemRouter chi.Router) {

		systemRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			u, _ := UserFromContext(ctx)
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
			SystemPage(r, u, sidebarGroups, c, v, containers).Render(r.Context(), w)
		})

		systemRouter.Get("/poll", func(w http.ResponseWriter, r *http.Request) {
			sse := datastar.NewSSE(w, r)

			// Watch for system updates
			ctx := r.Context()
			syswatcher, err := dm.Store.WatchAll(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer syswatcher.Stop()
			// docker container updates
			dockerwatcher, err := dm.Stores.DockerStore.Watch(ctx, "containers")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer dockerwatcher.Stop()

			// message updates
			messagewatcher, err := dm.Stores.MessageStore.WatchAll(ctx)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer messagewatcher.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-messagewatcher.Updates():
					//	slog.Info("Docker Update", "entry", entry)
					if entry == nil {
						continue
					}
					dm.Logger.Debug("update", "operation", entry.Operation())
					keys, err := dm.Stores.MessageStore.Keys(ctx)
					if err != nil {
						dm.Logger.Error("Message Update", "error", err)
					}
					var toasts []components.Toast
					for _, key := range keys {
						dm.Logger.Debug("key", "key", key)
						val, err := dm.Stores.MessageStore.Get(ctx, key)
						if err != nil {
							dm.Logger.Error("Message Update", "error", err)
						}
						var toast components.Toast
						if err := json.Unmarshal(val.Value(), &toast); err != nil {
							dm.Logger.Error("json unmarshal", "error", err)
							sse.ConsoleError(err)
							continue
						}
						toasts = append(toasts, toast)
						dm.BaseModule.ExpireToast(key, 10*time.Second)
					}
					c := components.ToastUpdate(toasts)

					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}

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
					c := docker.DockerOverviewCard(cc)
					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}
				case entry := <-syswatcher.Updates():
					//dm.Logger.Info("System Update", "entry", entry)
					if entry == nil {
						continue
					}
					switch k := entry.Key(); k {
					case "virtualMemory":
						//	slog.Info("Memory Update")
						var v mem.VirtualMemoryStat
						if err := json.Unmarshal(entry.Value(), &v); err != nil {
							slog.Error("Memory Update", "error", err)
							sse.ConsoleError(err)
							continue
						}
						memStats := SystemMonitorSignals{
							MemTotal:       ByteCountIEC(v.Total),
							MemUsed:        ByteCountIEC(v.Used),
							MemUsedPercent: strconv.FormatFloat(v.UsedPercent, 'f', 2, 64),
						}
						sse := datastar.NewSSE(w, r)
						sse.MarshalAndMergeSignals(memStats)

					case "services":
						//	slog.Info("CPU Update")
						var v []dbus.UnitStatus
						if err := json.Unmarshal(entry.Value(), &v); err != nil {
							slog.Error("Servic Update", "error", err)
							sse.ConsoleError(err)
							continue
						}
						c := services.ServiceOverviewCard(v)
						if err := sse.MergeFragmentTempl(c); err != nil {
							sse.ConsoleError(err)
							return
						}
					case "cpu":
						//	slog.Info("CPU Update")
						var v CPUSimple
						if err := json.Unmarshal(entry.Value(), &v); err != nil {
							slog.Error("CPU Update", "error", err)
							sse.ConsoleError(err)
							continue
						}
						// c := cpuDetailCard(v)
						// if err := sse.MergeFragmentTempl(c); err != nil {
						// 	sse.ConsoleError(err)
						// 	return
						// }
						cpuStats := SystemMonitorSignals{
							Cores:       strconv.Itoa(v.Cores),
							Usage:       v.Used,
							UsedPercent: v.UsedPercent,
						}
						sse := datastar.NewSSE(w, r)
						sse.MarshalAndMergeSignals(cpuStats)
					}
				}
			}
		})

	})
	return nil
}
