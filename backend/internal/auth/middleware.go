package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// resolver is the part of *UserResolver the middleware uses, so tests can
// substitute a stub without a database.
type resolver interface {
	Resolve(ctx context.Context, id Identity) (Principal, error)
}

// Authenticator rejects unauthenticated requests with 401 and attaches a
// Principal to everything it lets through.
//
// It answers authentication only. Whether a principal may perform a given
// mutation is decided in the handlers, where the owning team is known.
//
// Mount this on the /api/v1 sub-router: /healthz lives on the outer router and
// must stay public.
func Authenticator(cfg Config, v Verifier, users resolver, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			credential, ok := credentialFrom(r, cfg.Mode)
			if !ok {
				unauthorized(w, cfg.Mode, "missing credentials")
				return
			}

			ctx := r.Context()

			identity, err := v.Verify(ctx, credential)
			if err != nil {
				// Logged at debug and never echoed: the reason a token failed
				// is useful to an attacker and useless to a legitimate caller.
				logger.Debug("credential rejected", "error", err)
				unauthorized(w, cfg.Mode, "invalid credentials")
				return
			}

			principal, err := users.Resolve(ctx, identity)
			switch {
			case errors.Is(err, ErrUnknownUser):
				logger.Debug("unknown user", "email", identity.Email)
				unauthorized(w, cfg.Mode, "unknown user")
				return
			case err != nil:
				logger.Error("resolve principal", "error", err, "email", identity.Email)
				writeError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, principal)))
		})
	}
}

// credentialFrom extracts the caller's credential for the configured mode.
//
// The dev header is read only in dev mode. In Entra mode it is not consulted at
// all, so a caller cannot impersonate anyone by setting it — this is the
// security invariant of the whole design.
func credentialFrom(r *http.Request, mode Mode) (string, bool) {
	if mode == ModeDev {
		value := strings.TrimSpace(r.Header.Get(DevUserHeader))
		return value, value != ""
	}

	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	return token, token != ""
}

func unauthorized(w http.ResponseWriter, mode Mode, message string) {
	if mode == ModeEntra {
		w.Header().Set("WWW-Authenticate", "Bearer")
	}
	writeError(w, http.StatusUnauthorized, "unauthenticated", message)
}

// writeError emits the Error schema by hand. This middleware runs below the
// generated API layer and has no access to its response types.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}{Code: code, Message: message})
}
