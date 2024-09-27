package gophmarkthttpserver

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type contextKey int

const (
	contextAuthUser contextKey = iota
)

func (h *HTTPServer) newRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.StripSlashes)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(Logging(h.logger))

	// ping handler.
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})

	// user registration
	r.Post("/api/user/register", h.userRegistration)
	// user authentication
	r.Post("/api/user/login", h.userLogin)

	// handlers for authenticated users
	r.Group(func(r chi.Router) {
		r.Use(h.authUserCtx)
		// uploading the order number for calculation
		r.Post("/api/user/orders", h.ordersPut)
		// getting a list of orders uploaded by the user
		r.Get("/api/user/orders", h.ordersGet)
		// getting a balance by the user
		r.Get("/api/user/balance", h.balanceGet)
		// uploading the order for drawal
		r.Post("/api/user/balance/withdraw", h.drawalsPut)
		// getting a list of drawal orders uploaded by the user
		r.Get("/api/user/withdrawals", h.drawalsGet)
	})

	// stubs.
	r.Get("/*", notImplementedYet)
	r.Post("/*", notImplementedYet)
	r.Put("/*", notImplementedYet)

	return r
}

// not implemented handlers.
func notImplementedYet(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not implemented yet"))
}
