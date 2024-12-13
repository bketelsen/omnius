package incus

import (
	"context"
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

func (dm *IncusModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up Incus Routes")

	r.Route("/incus", func(incusRouter chi.Router) {

		incusRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			u, _ := UserFromContext(ctx)
			IncusPage(r, u, sidebarGroups).Render(r.Context(), w)
		})

		incusRouter.Get("/api", func(w http.ResponseWriter, r *http.Request) {

			//sse := datastar.NewSSE(w, r)
			incuswatcher, err := dm.Stores.IncusStore.Watch(ctx, "containers")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer incuswatcher.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-incuswatcher.Updates():
					dm.Logger.Info("Incus Update", "entry", entry)
					// if entry == nil {
					// 	continue
					// }
					// var cc []types.Container
					// if err := json.Unmarshal(entry.Value(), &cc); err != nil {
					// 	dm.Logger.Error("Incus Update", "error", err)
					// 	sse.ConsoleError(err)
					// 	continue
					// }
					// c := IncusOverviewCard(cc)
					// if err := sse.MergeFragmentTempl(c); err != nil {
					// 	sse.ConsoleError(err)
					// 	return
					// }

				}
			}

		})

	})
	return nil
}
