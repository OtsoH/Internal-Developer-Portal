package app

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/OtsoH/internal-developer-portal/backend/internal/auth"
)

// Built with a nil Queries, so the authenticator is not mounted and these tests
// exercise the validator on its own. That is the point: the request never gets
// far enough for a database to matter.
func testRouter(t *testing.T) http.Handler {
	t.Helper()
	h, err := NewRouter(t.Context(), Deps{Auth: auth.Config{Mode: auth.ModeDev}})
	require.NoError(t, err)
	return h
}

func do(t *testing.T, h http.Handler, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeError asserts the response carries components/schemas/Error as JSON.
// Every rejection has to keep that shape: the generated TypeScript client
// parses it, and the middleware's own default handler emits text/plain.
func decodeError(t *testing.T, rec *httptest.ResponseRecorder) map[string]string {
	t.Helper()
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "body was %q", rec.Body.String())
	var payload map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.NotEmpty(t, payload["code"])
	require.NotEmpty(t, payload["message"])
	return payload
}

// The probe sits on the outer router, above the API sub-router, so validation
// must not reach it — /healthz is not in the spec and would 404 if it did.
func TestHealthzBypassesValidation(t *testing.T) {
	rec := do(t, testRouter(t), http.MethodGet, "/healthz", nil)
	require.Equal(t, http.StatusOK, rec.Code)
}

// Guards the trap that makes this middleware look broken: clear spec.Servers and
// the validator looks for "/services" while chi delivers "/api/v1/services", so
// every valid request 404s.
func TestValidRequestsPassThrough(t *testing.T) {
	h := testRouter(t)
	for _, target := range []string{
		"/api/v1/services",
		"/api/v1/services?lifecycle=beta",
		"/api/v1/services?team=platform&tag=go&q=search",
		"/api/v1/teams",
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		require.Equal(t, http.StatusOK, rec.Code, "GET %s: %s", target, rec.Body.String())
	}
}

func TestQueryParameterOutsideEnumIsRejected(t *testing.T) {
	rec := do(t, testRouter(t), http.MethodGet, "/api/v1/services?lifecycle=bogus", nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	payload := decodeError(t, rec)
	require.Equal(t, "bad_request", payload["code"])
	require.Contains(t, payload["message"], "lifecycle")
	// One line per violation. kin-openapi's raw text restates the schema and the
	// value over a dozen lines; a caller holding the spec needs none of it.
	require.NotContains(t, payload["message"], "\n")
}

func TestMalformedPathParameterIsRejected(t *testing.T) {
	rec := do(t, testRouter(t), http.MethodGet, "/api/v1/services/not-a-uuid", nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "bad_request", decodeError(t, rec)["code"])
}

// The behaviour this step exists for: schema constraints the generated code
// ignores are now enforced, and all of them are reported at once.
func TestBodyConstraintsAreEnforced(t *testing.T) {
	body := `{"name":"` + strings.Repeat("x", 5000) + `","slug":"Not A Slug",` +
		`"teamId":"11111111-1111-1111-1111-111111111111","lifecycle":"bogus"}`
	rec := do(t, testRouter(t), http.MethodPost, "/api/v1/services", strings.NewReader(body))

	require.Equal(t, http.StatusBadRequest, rec.Code)
	message := decodeError(t, rec)["message"]
	require.Contains(t, message, "name")
	require.Contains(t, message, "slug")
	require.Contains(t, message, "lifecycle")
	// The summary must not hand the caller's own 5000-character payload back.
	require.Less(t, len(message), 1000, "message: %s", message)
}

func TestMissingRequiredBodyIsRejected(t *testing.T) {
	rec := do(t, testRouter(t), http.MethodPost, "/api/v1/services", nil)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "bad_request", decodeError(t, rec)["code"])
}

// An unknown path under /api/v1 is answered by the validator rather than chi,
// which would emit a bare text/plain "404 page not found".
func TestUnknownEndpointIsJSON(t *testing.T) {
	rec := do(t, testRouter(t), http.MethodGet, "/api/v1/nope", nil)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Equal(t, "not_found", decodeError(t, rec)["code"])
}
