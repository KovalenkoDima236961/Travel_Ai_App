package copilot

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestRegisterRoutesCoexistsWithTripRouter(t *testing.T) {
	router := chi.NewRouter()
	router.Route("/trips", func(r chi.Router) {
		r.Get("/", func(http.ResponseWriter, *http.Request) {})
	})

	(&Handler{}).RegisterRoutes(router)
}
