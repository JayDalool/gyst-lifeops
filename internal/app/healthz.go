package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

// healthzTimeout bounds the database probe so a wedged database reports
// unhealthy instead of hanging the caller.
const healthzTimeout = time.Second

// healthzHandler reports only that the process is serving and the database
// answers. It deliberately says nothing else: no version, build, path, driver
// or error text in the body or headers, and it does not log, because it is
// polled frequently.
//
// It is registered on the mux in front of the session middleware (see New), not
// added to the handlers package's auth-exempt list, so that list stays limited
// to routes that carry their own credential.
func healthzHandler(database *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")

		ctx, cancel := context.WithTimeout(r.Context(), healthzTimeout)
		defer cancel()
		var one int
		if err := database.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method == http.MethodHead {
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
}
