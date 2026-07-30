package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

// failingServer shadows ListServices with one that fails the way a real
// Postgres error arrives at the handler boundary.
type failingServer struct{ *Server }

const leakySQLError = `ERROR: relation "services" does not exist (SQLSTATE 42P01)`

func (failingServer) ListServices(context.Context, ListServicesRequestObject) (ListServicesResponseObject, error) {
	return nil, errors.New(leakySQLError)
}

// newTestServer wires the same handler stack main.go builds, over a logger the
// test can read back.
func newTestServer(t *testing.T, ssi StrictServerInterface) (*httptest.Server, *bytes.Buffer) {
	t.Helper()

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))

	strict := NewStrictHandlerWithOptions(ssi, nil, StrictOptions(logger))
	h := HandlerWithOptions(strict, ChiServerOptions{
		BaseRouter:       chi.NewRouter(),
		ErrorHandlerFunc: ChiErrorHandler,
	})

	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)
	return ts, &logs
}

func decodeError(t *testing.T, resp *http.Response) Error {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, "application/json", resp.Header.Get("Content-Type"),
		"error responses must match the Error schema the spec promises")

	var body Error
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	return body
}

// A handler error must never reach the caller: the response says nothing, the
// log says everything.
func TestHandlerErrorIsNotLeakedToClient(t *testing.T) {
	ts, logs := newTestServer(t, failingServer{NewServer(nil)})

	resp, err := http.Get(ts.URL + "/services")
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	body := decodeError(t, resp)
	require.Equal(t, "internal", body.Code)
	require.NotContains(t, body.Message, "SQLSTATE")
	require.NotContains(t, body.Message, "relation")

	// Substring rather than the whole message: the JSON handler escapes the
	// quotes around "services".
	require.Contains(t, logs.String(), "SQLSTATE 42P01",
		"the real error must still be recoverable from the log")
}

// Parameter binding failures are the caller's fault and describable, but they
// still have to be JSON.
func TestBadPathParamReturnsJSONBadRequest(t *testing.T) {
	ts, _ := newTestServer(t, NewServer(nil))

	resp, err := http.Get(ts.URL + "/services/not-a-uuid")
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeError(t, resp)
	require.Equal(t, "bad_request", body.Code)
	require.Contains(t, body.Message, "serviceId")
}

func TestMalformedJSONBodyReturnsJSONBadRequest(t *testing.T) {
	ts, _ := newTestServer(t, NewServer(nil))

	resp, err := http.Post(ts.URL+"/services", "application/json", strings.NewReader(`{"name":`))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeError(t, resp)
	require.Equal(t, "bad_request", body.Code)
}
