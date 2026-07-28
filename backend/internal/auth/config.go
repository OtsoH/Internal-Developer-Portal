package auth

import (
	"fmt"
	"os"
)

// Mode selects how callers prove who they are.
type Mode string

const (
	// ModeDev trusts an X-Dev-User header naming a seeded user. Local only.
	ModeDev Mode = "dev"
	// ModeEntra verifies a Microsoft Entra JWT.
	ModeEntra Mode = "entra"
)

// Config is the authentication configuration read from the environment.
type Config struct {
	Mode Mode
	// Issuer is the OIDC issuer URL, e.g.
	// https://<tenant>.ciamlogin.com/<tenantId>/v2.0
	Issuer string
	// Audience must match the aud claim of the tokens this API accepts —
	// the backend app registration, not Microsoft Graph.
	Audience string
}

// ConfigFromEnv reads AUTH_MODE, OIDC_ISSUER and OIDC_AUDIENCE.
//
// An unset AUTH_MODE means dev, so a bare `go run ./cmd/api` still works. Any
// value that is neither "dev" nor "entra" is an error rather than a silent
// fallback: a typo must never quietly downgrade a deployment to header trust.
func ConfigFromEnv() (Config, error) {
	mode := Mode(os.Getenv("AUTH_MODE"))
	if mode == "" {
		mode = ModeDev
	}

	cfg := Config{
		Mode:     mode,
		Issuer:   os.Getenv("OIDC_ISSUER"),
		Audience: os.Getenv("OIDC_AUDIENCE"),
	}

	switch mode {
	case ModeDev:
		return cfg, nil
	case ModeEntra:
		if cfg.Issuer == "" {
			return Config{}, fmt.Errorf("AUTH_MODE=entra requires OIDC_ISSUER")
		}
		if cfg.Audience == "" {
			return Config{}, fmt.Errorf("AUTH_MODE=entra requires OIDC_AUDIENCE")
		}
		return cfg, nil
	default:
		return Config{}, fmt.Errorf("unknown AUTH_MODE %q: want %q or %q", mode, ModeDev, ModeEntra)
	}
}
