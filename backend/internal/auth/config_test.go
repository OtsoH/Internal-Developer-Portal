package auth

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	const (
		issuer   = "https://contoso.ciamlogin.com/tenant-id/v2.0"
		audience = "api://backend-client-id"
	)

	t.Run("unset AUTH_MODE defaults to dev", func(t *testing.T) {
		t.Setenv("AUTH_MODE", "")
		cfg, err := ConfigFromEnv()
		require.NoError(t, err)
		require.Equal(t, ModeDev, cfg.Mode)
	})

	t.Run("dev mode needs no issuer or audience", func(t *testing.T) {
		t.Setenv("AUTH_MODE", "dev")
		t.Setenv("OIDC_ISSUER", "")
		t.Setenv("OIDC_AUDIENCE", "")
		cfg, err := ConfigFromEnv()
		require.NoError(t, err)
		require.Equal(t, ModeDev, cfg.Mode)
	})

	t.Run("entra mode reads issuer and audience", func(t *testing.T) {
		t.Setenv("AUTH_MODE", "entra")
		t.Setenv("OIDC_ISSUER", issuer)
		t.Setenv("OIDC_AUDIENCE", audience)
		cfg, err := ConfigFromEnv()
		require.NoError(t, err)
		require.Equal(t, Config{Mode: ModeEntra, Issuer: issuer, Audience: audience}, cfg)
	})

	t.Run("entra mode without an issuer fails at startup", func(t *testing.T) {
		t.Setenv("AUTH_MODE", "entra")
		t.Setenv("OIDC_ISSUER", "")
		t.Setenv("OIDC_AUDIENCE", audience)
		_, err := ConfigFromEnv()
		require.ErrorContains(t, err, "OIDC_ISSUER")
	})

	t.Run("entra mode without an audience fails at startup", func(t *testing.T) {
		t.Setenv("AUTH_MODE", "entra")
		t.Setenv("OIDC_ISSUER", issuer)
		t.Setenv("OIDC_AUDIENCE", "")
		_, err := ConfigFromEnv()
		require.ErrorContains(t, err, "OIDC_AUDIENCE")
	})

	// A typo must fail loudly rather than quietly downgrading a deployment to
	// header-based trust.
	t.Run("an unrecognized mode is an error, not a fallback", func(t *testing.T) {
		t.Setenv("AUTH_MODE", "entraa")
		_, err := ConfigFromEnv()
		require.ErrorContains(t, err, "unknown AUTH_MODE")
	})
}
