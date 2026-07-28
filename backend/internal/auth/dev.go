package auth

import (
	"context"
	"strings"
)

// DevUserHeader names the user in AUTH_MODE=dev. The value is a bare email that
// must already exist in the users table.
//
// This header is read only when Config.Mode is ModeDev. It is a
// trust-the-network shortcut for local development: anything able to reach the
// backend directly can assert any identity with it, so a backend running in dev
// mode must never be exposed.
const DevUserHeader = "X-Dev-User"

// DevVerifier accepts an email as the whole credential. It asserts only who the
// caller claims to be; UserResolver still rejects anyone not already seeded.
type DevVerifier struct{}

func (DevVerifier) Verify(_ context.Context, raw string) (Identity, error) {
	email := normalizeEmail(raw)
	if email == "" || !strings.Contains(email, "@") {
		return Identity{}, ErrInvalidToken
	}
	// Name is left empty on purpose: the resolver reads it from the seeded row
	// rather than letting a header rename a user.
	return Identity{Email: email}, nil
}

// normalizeEmail lowercases and trims an address so that lookups and inserts
// agree. The users_email_key index rules out a functional lower(email)
// comparison in SQL, so normalization has to happen here.
func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
