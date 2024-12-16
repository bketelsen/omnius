package web

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/bketelsen/omnius/web/layouts"
	datastar "github.com/starfederation/datastar/sdk/go"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/sessions"
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

func setupAuthRoutes(r chi.Router, sessionStore sessions.Store, sidebarGroups []*layouts.SidebarGroup) {

	r.Route("/auth", func(authRouter chi.Router) {
		authRouter.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
			sess, err := sessionStore.Get(r, "omnius")
			if err != nil {
				http.Error(w, "failed to get session", http.StatusInternalServerError)
				return
			}

			delete(sess.Values, "userID")
			if err := sess.Save(r, w); err != nil {
				http.Error(w, "failed to save session", http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/login", http.StatusUnauthorized)

		})

		authRouter.Route("/login", func(loginRouter chi.Router) {
			loginRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
				if _, ok := UserFromContext(r.Context()); ok {
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}

				PageAuthenticationLogin(r, "", sidebarGroups).Render(r.Context(), w)
			})

			loginRouter.Post("/", func(w http.ResponseWriter, r *http.Request) {
				type Form struct {
					Username string `json:"username"`
					Password string `json:"password"`
				}

				form := &Form{}
				if err := datastar.ReadSignals(r, form); err != nil {
					http.Error(w, "failed to parse request body", http.StatusBadRequest)
					return
				}

				slog.Info("form", slog.String("username", form.Username))

				// do the auth here
				// todo - what auth? ssh? PAM? creds from a file?
				if form.Username != "bjk" {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				// save the user in the session

				sess, err := sessionStore.Get(r, "omnius")
				if err != nil {
					http.Error(w, "failed to get session", http.StatusInternalServerError)
					return
				}

				sess.Values["userID"] = form.Username
				if err := sess.Save(r, w); err != nil {
					http.Error(w, "failed to save session", http.StatusInternalServerError)
					return
				}
				slog.Info("redirecting", slog.String("username", form.Username))
				sse := datastar.NewSSE(w, r)
				sse.ExecuteScript("window.location = \"/overview\"")

			})
		})

	})
}
