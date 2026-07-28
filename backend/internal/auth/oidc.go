package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// OIDCVerifier validates Entra-issued JWTs against the tenant's public keys.
type OIDCVerifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewOIDCVerifier performs OIDC discovery against the issuer once and returns a
// verifier that reuses the resulting key set.
//
// Construct this exactly once at startup, never per request: the underlying
// RemoteKeySet caches the JWKS and only refetches when it sees an unknown key
// id. Discovery is a network call, so an unreachable tenant fails the process
// at boot rather than on the first request — the same philosophy as running
// migrations at startup.
func NewOIDCVerifier(ctx context.Context, issuer, audience string) (*OIDCVerifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery for %s: %w", issuer, err)
	}
	return &OIDCVerifier{
		verifier: provider.Verifier(&oidc.Config{
			// ClientID is the aud check. If tokens arrive addressed to Microsoft
			// Graph instead of this API, that means the frontend requested the
			// wrong scope — see docs/entra-setup.md.
			ClientID: audience,
			// Clock skew knob, if Azure time drift ever becomes a problem:
			// Now: func() time.Time { return time.Now().Add(-30 * time.Second) }
			Now: time.Now,
		}),
	}, nil
}

func (o *OIDCVerifier) Verify(ctx context.Context, raw string) (Identity, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Identity{}, ErrInvalidToken
	}

	token, err := o.verifier.Verify(ctx, raw)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	var claims struct {
		OID               string `json:"oid"`
		Email             string `json:"email"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
	}
	if err := token.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("%w: decode claims: %v", ErrInvalidToken, err)
	}

	// Entra External ID puts the stable user id in oid; sub is per-application
	// and would change if the app registration were recreated.
	oid := claims.OID
	if oid == "" {
		oid = token.Subject
	}

	email := claims.Email
	if email == "" {
		email = claims.PreferredUsername
	}
	email = normalizeEmail(email)

	if oid == "" || email == "" {
		return Identity{}, fmt.Errorf("%w: token has no usable subject or email", ErrInvalidToken)
	}

	name := strings.TrimSpace(claims.Name)
	if name == "" {
		name = email
	}

	return Identity{EntraOID: oid, Email: email, Name: name}, nil
}
