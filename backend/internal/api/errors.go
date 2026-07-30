package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// The generated error handlers write err.Error() into the response body as
// text/plain. Two problems: a 500 hands the caller whatever the failure said —
// for a database error that is table names, constraint names and SQLSTATE — and
// every error response violates the Error schema the spec promises, so the
// generated TypeScript client cannot parse it.
//
// The split below: a 4xx is the caller's fault and safe to describe, because the
// only things those messages contain are spec-declared parameter names (already
// public) and the caller's own payload. A 5xx is our fault and stays opaque —
// the detail goes to the log, correlated by request id.

// StrictOptions supplies the error handlers for NewStrictHandlerWithOptions:
// RequestErrorHandlerFunc fires when a JSON body fails to decode,
// ResponseErrorHandlerFunc when a handler returns a non-nil error.
func StrictOptions(logger *slog.Logger) StrictHTTPServerOptions {
	if logger == nil {
		logger = slog.Default()
	}
	return StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, err error) {
			writeAPIError(w, http.StatusBadRequest, "bad_request", err.Error())
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			internalError(w, r, logger, err)
		},
	}
}

// ChiErrorHandler handles path- and query-parameter binding failures. Those are
// raised by the generated router before the strict layer is reached, so they
// need their own hook — StrictOptions never sees them.
func ChiErrorHandler(w http.ResponseWriter, _ *http.Request, err error) {
	writeAPIError(w, http.StatusBadRequest, "bad_request", err.Error())
}

func internalError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	requestID := middleware.GetReqID(r.Context())

	logger.Error("unhandled error serving request",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)

	// A VisitXResponse can fail after it has already written the status line —
	// an encoding error partway through the body, say. Writing a second header
	// there yields "superfluous WriteHeader" and a corrupt response, so once the
	// response has started the log above is the only thing left to do.
	if ww, ok := w.(middleware.WrapResponseWriter); ok && ww.Status() != 0 {
		return
	}

	message := "Something went wrong."
	if requestID != "" {
		message = fmt.Sprintf("Something went wrong. Request id: %s", requestID)
	}
	writeAPIError(w, http.StatusInternalServerError, "internal", message)
}

// writeAPIError emits components/schemas/Error. Named to stay distinct from the
// hand-rolled writeError in package auth, which runs below this layer and has no
// access to the generated types.
func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Error{Code: code, Message: message})
}
