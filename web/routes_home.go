package web

import (
	"net/http"

	"github.com/delaneyj/toolbelt/embeddednats"
	"github.com/go-chi/chi/v5"
)

func setupHomeRoutes(r chi.Router, _ *embeddednats.Server) error {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		HomePage().Render(r.Context(), w)

	})

	return nil
}
