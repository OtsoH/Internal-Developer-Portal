package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// RequestValidator returns middleware that rejects any request the embedded
// OpenAPI spec does not allow, before it reaches a handler.
//
// oapi-codegen generates no validation of its own. It binds parameters and
// decodes bodies, but `minLength`, `maxLength`, `pattern`, `enum` and `format`
// are all decoration as far as the generated code is concerned: today a
// 5000-character name or `?lifecycle=bogus` travels straight through. The spec
// already states every one of those limits, so enforcing them from the spec
// itself is the difference between documentation and a contract.
//
// It returns an error rather than panicking so a malformed spec fails the
// process at boot, alongside the other startup checks.
func RequestValidator(logger *slog.Logger) (func(http.Handler) http.Handler, error) {
	if logger == nil {
		logger = slog.Default()
	}

	spec, err := GetSpec()
	if err != nil {
		return nil, fmt.Errorf("loading embedded OpenAPI spec: %w", err)
	}

	// By default every kin-openapi schema error appends a "Schema:" and a
	// "Value:" block: the offending subschema as JSON, then the caller's own
	// submitted value. That is a dozen lines per violation, it echoes the whole
	// payload back, and its newlines hide the other violations behind the first
	// line. The flag is package-global and process-wide, which is the only knob
	// the library offers; setting it here keeps it next to the reason.
	openapi3.SchemaErrorDetailsDisabled = true

	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			// The spec declares a global `security` requirement and kin-openapi
			// enforces it. Authentication has already happened in
			// auth.Authenticator, mounted above this middleware, which knows
			// about both AUTH_MODE=dev (X-Dev-User) and Entra (Bearer);
			// kin-openapi knows only the latter. Without this no-op, every
			// dev-mode request 401s here with "security requirements failed".
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
			// Report every violation in one response. A form submitting three
			// bad fields should not need three round trips to learn that.
			MultiError: true,
		},
		// spec.Servers stays set, deliberately. chi's Mount rewrites the route
		// context, not r.URL.Path, so this middleware still sees the full
		// "/api/v1/services"; the gorillamux router underneath builds its routes
		// as server URL + path, which reproduces exactly that prefix. Clearing
		// Servers makes every request 404 with "no matching operation was
		// found". The warning being silenced here is about Host-header
		// validation, which cannot apply: the server URL is a bare path with no
		// host to match against.
		SilenceServersWarning: true,
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			validationError(w, r, logger, err, opts.StatusCode)
		},
	}), nil
}

// validationError renders a rejected request as the Error schema, following the
// same split as the generated-handler hooks in errors.go: a 4xx describes what
// the caller got wrong, a 5xx stays opaque with the detail in the log.
func validationError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error, status int) {
	switch status {
	case http.StatusBadRequest:
		writeAPIError(w, status, "bad_request", summarizeValidationError(err))
	case http.StatusNotFound:
		// The path matched /api/v1 but no operation in the spec. Answering from
		// here rather than from chi keeps unknown endpoints on the Error schema.
		writeAPIError(w, status, "not_found", "no such endpoint")
	case http.StatusUnauthorized:
		// Unreachable while AuthenticationFunc is the no-op above, but the
		// middleware can still produce this status, so it gets a real answer
		// instead of falling into the 500 branch.
		writeAPIError(w, status, "unauthenticated", "not authenticated")
	default:
		internalError(w, r, logger, err)
	}
}

// summarizeValidationError renders every violation on one line. With schema
// detail disabled, kin-openapi already joins a MultiError with " | " and each
// entry is a single sentence, so this only has to guarantee the invariant for
// any error type that still carries a newline.
func summarizeValidationError(err error) string {
	return strings.ReplaceAll(err.Error(), "\n", " ")
}
