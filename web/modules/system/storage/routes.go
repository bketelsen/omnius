package storage

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
func (dm *StorageModule) SetupRoutes(r chi.Router, sidebarGroups []*layouts.SidebarGroup, ctx context.Context) error {
	dm.Logger.Info("Setting up Storage Routes")

	r.Route("/storage", func(storageRouter chi.Router) {

		storageRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			u, _ := UserFromContext(ctx)
			StoragePage(r, u, sidebarGroups).Render(r.Context(), w)
		})

	})
	return nil
}
