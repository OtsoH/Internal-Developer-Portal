package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDevVerifier(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		wantEmail string
		wantErr   bool
	}{
		{name: "plain email", raw: "dev.admin@example.com", wantEmail: "dev.admin@example.com"},
		{name: "normalizes case and padding", raw: "  Dev.Admin@Example.COM  ", wantEmail: "dev.admin@example.com"},
		{name: "empty", raw: "", wantErr: true},
		{name: "whitespace only", raw: "   ", wantErr: true},
		{name: "not an email", raw: "dev-admin", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := DevVerifier{}.Verify(context.Background(), tt.raw)
			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidToken)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantEmail, id.Email)
			require.Empty(t, id.EntraOID, "dev mode asserts no object id")
		})
	}
}
