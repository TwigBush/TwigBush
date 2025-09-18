package playground

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func MountUI(r chi.Router) {
	fs := http.FileServer(http.Dir("web/playground"))
	r.Handle("/playground/*", http.StripPrefix("/playground/", fs))
	r.Get("/playground", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/playground/index.html", http.StatusFound)
	})
}
