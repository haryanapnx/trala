package router

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	swgui "github.com/swaggest/swgui/v5emb"

	docs "github.com/haryanapnx/trala/docs/swagger"
	"github.com/haryanapnx/trala/internal/domain"
	"github.com/haryanapnx/trala/internal/transport/http/handler"
	"github.com/haryanapnx/trala/internal/transport/http/middleware"
	"github.com/haryanapnx/trala/internal/transport/http/response"
)

type Deps struct {
	Logger   *slog.Logger
	Health   *handler.HealthHandler
	Auth     *handler.AuthHandler
	Status   *handler.StatusHandler
	Tasks    *handler.TaskHandler
	Verifier middleware.TokenVerifier
}

// New builds the HTTP router: global middleware, unversioned health probes, the
// public /v1/auth routes, and the authenticated /v1 routes.
func New(deps Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger(deps.Logger))
	r.Use(middleware.Recover(deps.Logger))

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, r, domain.ErrNotFound.WithMessage("route not found"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		response.Error(w, r, domain.New(domain.CodeMethodNotAllowed, "method not allowed"))
	})

	// Health probes are intentionally unversioned.
	r.Get("/healthz", deps.Health.Live)
	r.Get("/readyz", deps.Health.Ready)

	// Swagger UI. The spec is served from the generated docs package; static
	// assets are embedded, so /docs works without network access.
	r.Get("/docs/swagger.json", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(docs.SwaggerInfo.ReadDoc()))
	})
	swaggerUI := swgui.New("Task Management API", "/docs/swagger.json", "/docs")
	r.Handle("/docs", swaggerUI)
	r.Handle("/docs/*", swaggerUI)

	r.Route("/v1", func(r chi.Router) {
		// Public.
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", deps.Auth.Register)
			r.Post("/login", deps.Auth.Login)
		})

		// Authenticated.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(deps.Verifier))

			r.Route("/statuses", func(r chi.Router) {
				r.Get("/", deps.Status.List)
				r.Post("/", deps.Status.Create)
				r.Put("/{id}", deps.Status.Update)
				r.Delete("/{id}", deps.Status.Delete)
			})

			r.Route("/tasks", func(r chi.Router) {
				r.Post("/", deps.Tasks.Create)
				r.Get("/", deps.Tasks.List)
				r.Get("/{id}", deps.Tasks.Get)
				r.Put("/{id}", deps.Tasks.Update)
				r.Delete("/{id}", deps.Tasks.Delete)
				r.Post("/{id}/assign", deps.Tasks.Assign)
			})
		})
	})

	return r
}
