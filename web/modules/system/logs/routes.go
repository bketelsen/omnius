package logs

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bketelsen/omnius/web/layouts"
	"github.com/go-chi/chi/v5"
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
func (dm *LogModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up Log Routes")

	r.Route("/logs", func(logRouter chi.Router) {

		logRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			u, _ := UserFromContext(ctx)
			LogPage(r, u, sidebarGroups).Render(r.Context(), w)
		})

		logRouter.Get("/api", func(w http.ResponseWriter, r *http.Request) {

			//		sse := datastar.NewSSE(w, r)
			logwatcher, err := dm.Stores.DockerStore.Watch(ctx, "containers")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer logwatcher.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-logwatcher.Updates():
					slog.Info("Docker Update", "entry", entry)

				}
			}

		})
	})
	return nil
}
