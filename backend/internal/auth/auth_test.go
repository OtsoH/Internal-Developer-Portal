package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRoleAtLeast(t *testing.T) {
	tests := []struct {
		name string
		have Role
		min  Role
		want bool
	}{
		{"admin satisfies admin", RoleAdmin, RoleAdmin, true},
		{"admin satisfies editor", RoleAdmin, RoleEditor, true},
		{"admin satisfies viewer", RoleAdmin, RoleViewer, true},
		{"editor satisfies editor", RoleEditor, RoleEditor, true},
		{"editor satisfies viewer", RoleEditor, RoleViewer, true},
		{"editor does not satisfy admin", RoleEditor, RoleAdmin, false},
		{"viewer satisfies viewer", RoleViewer, RoleViewer, true},
		{"viewer does not satisfy editor", RoleViewer, RoleEditor, false},
		{"viewer does not satisfy admin", RoleViewer, RoleAdmin, false},
		// The zero Role is what a missing membership looks like, so it must
		// fail every requirement including the weakest one.
		{"no membership does not satisfy viewer", Role(""), RoleViewer, false},
		{"no membership does not satisfy admin", Role(""), RoleAdmin, false},
		{"unrecognized role satisfies nothing", Role("OWNER"), RoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.have.AtLeast(tt.min))
		})
	}
}

func TestPrincipalRoleLookups(t *testing.T) {
	platform := uuid.New()
	payments := uuid.New()
	unrelated := uuid.New()

	p := Principal{
		UserID: uuid.New(),
		Email:  "dev.editor@example.com",
		Roles: map[uuid.UUID]TeamRole{
			platform: {TeamID: platform, TeamSlug: "platform", TeamName: "Platform", Role: RoleViewer},
			payments: {TeamID: payments, TeamSlug: "payments", TeamName: "Payments", Role: RoleEditor},
		},
	}

	require.Equal(t, RoleViewer, p.RoleIn(platform))
	require.Equal(t, RoleEditor, p.RoleIn(payments))
	require.Equal(t, Role(""), p.RoleIn(unrelated), "a team with no membership yields the zero role")

	// Membership without power: a viewer on the team still cannot write to it.
	require.False(t, p.HasRoleIn(platform, RoleEditor))
	require.True(t, p.HasRoleIn(payments, RoleEditor))
	require.False(t, p.HasRoleIn(payments, RoleAdmin))
	require.False(t, p.HasRoleIn(unrelated, RoleViewer))
}

func TestPrincipalContextRoundTrip(t *testing.T) {
	want := Principal{UserID: uuid.New(), Email: "dev.admin@example.com"}

	got, ok := PrincipalFrom(WithPrincipal(context.Background(), want))
	require.True(t, ok)
	require.Equal(t, want, got)

	_, ok = PrincipalFrom(context.Background())
	require.False(t, ok, "a bare context carries no principal")
}
