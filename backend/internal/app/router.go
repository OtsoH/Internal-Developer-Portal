// Package app assembles the HTTP handler from its parts.
//
// It exists as its own package so that tests can exercise the production router
// — the real middleware chain, in the real order — rather than a reassembled
// lookalike that can drift from what main.go actually serves.
package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/OtsoH/internal-developer-portal/backend/internal/api"
	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
	dbgen "github.com/OtsoH/internal-developer-portal/backend/internal/db/gen"
	"github.com/OtsoH/internal-developer-portal/backend/internal/httpx"
)

// Deps are everything the router needs from the process around it.
type Deps struct {
	// Queries is nil when DATABASE_URL is unset. That mode keeps /healthz and
	// the read endpoints alive, and leaves the API unauthenticated because the
	// user resolver has nothing to resolve against.
	Queries *dbgen.Queries
	// Tx is the transaction source for mutations. Nil alongside Queries.
	Tx     api.TxBeginner
	Auth   auth.Config
	Logger *slog.Logger
}

// NewRouter builds the full HTTP handler.
//
// It returns an error because Entra mode performs OIDC discovery here: an
// unreachable tenant must fail the process at boot rather than every request
// after it, the same reasoning as running migrations at startup.
func NewRouter(ctx context.Context, deps Deps) (http.Handler, error) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// middleware.RealIP is deliberately absent: it is deprecated as spoofable
	// (trusts X-Forwarded-For & co. unconditionally, GHSA-3fxj-6jh8-hvhx).
	// Logs record the direct peer address instead.
	r.Use(httpx.RequestLogger(logger))
	r.Use(middleware.Recoverer)

	// Health checks stay on the outer router, above the authenticator: a
	// platform probe has no credentials and must never need any.
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	apiRouter := chi.NewRouter()
	if deps.Queries != nil {
		verifier, err := newVerifier(ctx, deps.Auth, logger)
		if err != nil {
			return nil, err
		}
		// Registered before the generated routes: chi requires middleware to be
		// declared ahead of the handlers it wraps.
		apiRouter.Use(auth.Authenticator(deps.Auth, verifier, auth.NewUserResolver(deps.Queries), logger))
	} else {
		logger.Warn("no database configured: /api/v1 is unauthenticated and serves empty results")
	}

	server := api.NewServer(deps.Queries, api.WithTxBeginner(deps.Tx), api.WithLogger(logger))
	// The generated error handlers echo err.Error() to the caller as text/plain;
	// these replace both with the Error schema and keep 500 detail in the log.
	strict := api.NewStrictHandlerWithOptions(server, nil, api.StrictOptions(logger))
	r.Mount("/api/v1", api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:       apiRouter,
		ErrorHandlerFunc: api.ChiErrorHandler,
	}))

	return r, nil
}

func newVerifier(ctx context.Context, cfg auth.Config, logger *slog.Logger) (auth.Verifier, error) {
	switch cfg.Mode {
	case auth.ModeDev:
		logger.Warn("AUTH_MODE=dev: any caller may assert an identity with the " +
			auth.DevUserHeader + " header; never expose this backend")
		return auth.DevVerifier{}, nil
	case auth.ModeEntra:
		verifier, err := auth.NewOIDCVerifier(ctx, cfg.Issuer, cfg.Audience)
		if err != nil {
			return nil, err
		}
		logger.Info("authenticating against Entra", "issuer", cfg.Issuer, "audience", cfg.Audience)
		return verifier, nil
	default:
		// ConfigFromEnv rejects unknown modes already; this covers a Config
		// built by hand, and keeps the switch exhaustive.
		return nil, fmt.Errorf("unknown auth mode %q", cfg.Mode)
	}
}
