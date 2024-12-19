package incus

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bketelsen/omnius/web/components"
	"github.com/bketelsen/omnius/web/layouts"
	"github.com/go-chi/chi/v5"
	"github.com/lxc/incus/v6/shared/api"
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

func (dm *IncusModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up Incus Routes")

	r.Route("/incus", func(incusRouter chi.Router) {

		incusRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			u, _ := UserFromContext(ctx)
			instances, err := dm.client.GetInstancesAllProjects(api.InstanceTypeAny)
			if err != nil {
				dm.Logger.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return

			}
			images, err := dm.client.GetImagesAllProjects()
			if err != nil {
				dm.Logger.Error(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return

			}
			IncusPage(r, u, sidebarGroups, instances, images).Render(r.Context(), w)
		})

		incusRouter.Get("/api", func(w http.ResponseWriter, r *http.Request) {

			sse := datastar.NewSSE(w, r)
			incuswatcher, err := dm.Stores.IncusStore.Watch(ctx, "instances")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer incuswatcher.Stop()
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
				case entry := <-incuswatcher.Updates():
					dm.Logger.Debug("Incus Update", "entry", entry)
					if entry == nil {
						continue
					}
					var cc []api.Instance
					if err := json.Unmarshal(entry.Value(), &cc); err != nil {
						dm.Logger.Error("Incus Update", "error", err)
						sse.ConsoleError(err)
						continue
					}
					c := IncusDetail(cc)
					if err := sse.MergeFragmentTempl(c); err != nil {
						sse.ConsoleError(err)
						return
					}

				}
			}

		})
		incusRouter.Post("/api/{id}/stop", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			dm.Logger.Info("Incus Stopping", "instance", idParam)

			op, err := dm.client.UpdateInstanceState(idParam, api.InstanceStatePut{Action: "stop"}, "")
			if err != nil {
				dm.Logger.Info("Incus Stop", "error", err.Error())

				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = op.Wait()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			toast := components.Toast{
				Title:   "Instance Stopped",
				Message: idParam,
				Type:    components.AlertSuccess,
			}
			toastBytes, err := json.Marshal(toast)
			if err != nil {
				dm.Logger.Error(err.Error())
			} else {
				_, err = dm.Stores.MessageStore.Put(ctx, idParam, toastBytes)
				if err != nil {
					dm.Logger.Error(err.Error())
				}
			}

		})
		incusRouter.Post("/api/{id}/start", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			dm.Logger.Info("Incus Starting", "instance", idParam)

			op, err := dm.client.UpdateInstanceState(idParam, api.InstanceStatePut{Action: "start"}, "")
			if err != nil {
				dm.Logger.Info("Incus Start", "error", err.Error())

				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = op.Wait()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			toast := components.Toast{
				Title:   "Instance Started",
				Message: idParam,
				Type:    components.AlertSuccess,
			}
			toastBytes, err := json.Marshal(toast)
			if err != nil {
				dm.Logger.Error(err.Error())
			} else {
				_, err = dm.Stores.MessageStore.Put(ctx, idParam, toastBytes)
				if err != nil {
					dm.Logger.Error(err.Error())
				}
			}

		})
		incusRouter.Post("/api/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			dm.Logger.Info("Incus Pausing", "instance", idParam)

			op, err := dm.client.UpdateInstanceState(idParam, api.InstanceStatePut{Action: "freeze"}, "")
			if err != nil {
				dm.Logger.Info("Incus Freeze", "error", err.Error())

				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = op.Wait()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		})
		incusRouter.Post("/api/{id}/unpause", func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			dm.Logger.Info("Incus Unpausing", "instance", idParam)

			op, err := dm.client.UpdateInstanceState(idParam, api.InstanceStatePut{Action: "unfreeze"}, "")
			if err != nil {
				dm.Logger.Info("Incus Unfreeze", "error", err.Error())

				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			err = op.Wait()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		})

	})
	return nil
}
