package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// stubVerifier accepts one credential and rejects everything else.
type stubVerifier struct {
	accept   string
	identity Identity
}

func (s stubVerifier) Verify(_ context.Context, raw string) (Identity, error) {
	if raw != s.accept {
		return Identity{}, ErrInvalidToken
	}
	return s.identity, nil
}

// stubResolver returns a fixed principal for a known email.
type stubResolver struct {
	known map[string]Principal
	err   error
}

func (s stubResolver) Resolve(_ context.Context, id Identity) (Principal, error) {
	if s.err != nil {
		return Principal{}, s.err
	}
	p, ok := s.known[id.Email]
	if !ok {
		return Principal{}, ErrUnknownUser
	}
	return p, nil
}

const knownEmail = "dev.editor@example.com"

func testPrincipal() Principal {
	team := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	return Principal{
		UserID: uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000002"),
		Email:  knownEmail,
		Name:   "Dev Editor",
		Roles: map[uuid.UUID]TeamRole{
			team: {TeamID: team, TeamSlug: "platform", TeamName: "Platform", Role: RoleEditor},
		},
	}
}

// serve runs a request through the middleware and reports what the protected
// handler saw.
func serve(t *testing.T, cfg Config, v Verifier, res resolver, r *http.Request) (*httptest.ResponseRecorder, *Principal) {
	t.Helper()

	var seen *Principal
	protected := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if p, ok := PrincipalFrom(req.Context()); ok {
			seen = &p
		}
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	Authenticator(cfg, v, res, slog.New(slog.DiscardHandler))(protected).ServeHTTP(rec, r)
	return rec, seen
}

func requireErrorBody(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, wantCode, body.Code)
	require.NotEmpty(t, body.Message)
}

func devSetup() (Config, Verifier, resolver) {
	return Config{Mode: ModeDev},
		DevVerifier{},
		stubResolver{known: map[string]Principal{knownEmail: testPrincipal()}}
}

func TestDevModeWithoutHeaderIsUnauthorized(t *testing.T) {
	cfg, v, res := devSetup()

	rec, seen := serve(t, cfg, v, res, httptest.NewRequest(http.MethodGet, "/services", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Nil(t, seen)
	requireErrorBody(t, rec, "unauthenticated")
}

func TestDevModeWithUnknownUserIsUnauthorized(t *testing.T) {
	cfg, v, res := devSetup()

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	req.Header.Set(DevUserHeader, "nobody@example.com")
	rec, seen := serve(t, cfg, v, res, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Nil(t, seen)
	requireErrorBody(t, rec, "unauthenticated")
}

func TestDevModeWithSeededUserAttachesPrincipal(t *testing.T) {
	cfg, v, res := devSetup()

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	// Mixed case exercises normalization on the way through the verifier.
	req.Header.Set(DevUserHeader, "Dev.Editor@Example.com")
	rec, seen := serve(t, cfg, v, res, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, seen)
	require.Equal(t, knownEmail, seen.Email)
	require.True(t, seen.HasRoleIn(uuid.MustParse("11111111-1111-1111-1111-111111111111"), RoleEditor))
}

// The security invariant of the whole design: outside dev mode the dev header
// is not a credential, so it cannot be used to impersonate anyone.
func TestEntraModeIgnoresTheDevHeader(t *testing.T) {
	cfg := Config{Mode: ModeEntra, Issuer: "https://issuer.example", Audience: "api://backend"}
	v := stubVerifier{accept: "good-token", identity: Identity{EntraOID: "oid-1", Email: knownEmail}}
	res := stubResolver{known: map[string]Principal{knownEmail: testPrincipal()}}

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	req.Header.Set(DevUserHeader, "dev.admin@example.com")
	rec, seen := serve(t, cfg, v, res, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Nil(t, seen)
	require.Equal(t, "Bearer", rec.Header().Get("WWW-Authenticate"))
}

func TestEntraModeBearerTokens(t *testing.T) {
	cfg := Config{Mode: ModeEntra, Issuer: "https://issuer.example", Audience: "api://backend"}
	v := stubVerifier{accept: "good-token", identity: Identity{EntraOID: "oid-1", Email: knownEmail}}
	res := stubResolver{known: map[string]Principal{knownEmail: testPrincipal()}}

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{"valid token", "Bearer good-token", http.StatusOK},
		{"lowercase scheme", "bearer good-token", http.StatusOK},
		{"invalid token", "Bearer bad-token", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
		{"wrong scheme", "Basic good-token", http.StatusUnauthorized},
		{"scheme with no token", "Bearer ", http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/services", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec, seen := serve(t, cfg, v, res, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantStatus == http.StatusOK {
				require.NotNil(t, seen)
				require.Equal(t, knownEmail, seen.Email)
			} else {
				require.Nil(t, seen)
			}
		})
	}
}

func TestResolverFailureIsInternalAndLeaksNothing(t *testing.T) {
	cfg := Config{Mode: ModeDev}
	res := stubResolver{err: errors.New("connection refused to postgres://idp:secret@db:5432")}

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	req.Header.Set(DevUserHeader, knownEmail)
	rec, seen := serve(t, cfg, DevVerifier{}, res, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Nil(t, seen)
	requireErrorBody(t, rec, "internal")
	require.NotContains(t, rec.Body.String(), "postgres")
	require.NotContains(t, rec.Body.String(), "secret")
}
