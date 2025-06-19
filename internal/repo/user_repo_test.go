package repo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/talx-hub/gopher-bonus/internal/model/user"
)

func TestUserRepository_Create(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewUserRepository)
	defer cancel()

	tests := []struct {
		name       string
		loginHash  string
		password   string
		wantExists bool
		wantErr    bool
	}{
		{"create user1", "user1hash", "user1password-hash", true, false},
		{"create user2", "user2hash", "user2password-hash", true, false},
		{"duplicate login", "user1hash", "another-password", true, true},
		{"empty login", "", "some-password", false, true},
		{"empty password", "some-new-user", "", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.Create(ctx, &user.User{
				LoginHash:    tt.loginHash,
				PasswordHash: tt.password,
			})

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			exists := repo.Exists(ctx, tt.loginHash)
			assert.Equal(t, tt.wantExists, exists)
		})
	}
}

func TestUserRepository_FindByLogin(t *testing.T) {
	repo, ctx, cancel, _ := setupRepo(t, NewUserRepository)
	defer cancel()

	tests := []struct {
		name      string
		loginHash string
		wantUser  user.User
		wantErr   bool
	}{
		{
			name:      "existing user",
			loginHash: "user1hash",
			wantUser: user.User{
				LoginHash:    "user1hash",
				PasswordHash: "user1password-hash",
			},
			wantErr: false,
		},
		{
			name:      "non-existing user",
			loginHash: "no-such-user",
			wantUser:  user.User{},
			wantErr:   true,
		},
		{
			name:      "empty login",
			loginHash: "",
			wantUser:  user.User{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := repo.FindByLogin(ctx, tt.loginHash)
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, user.User{}, u)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantUser.LoginHash, u.LoginHash)
				assert.Equal(t, tt.wantUser.PasswordHash, u.PasswordHash)
				assert.NotEmpty(t, u.ID)
			}
		})
	}
}

func TestUserRepository_FindByID(t *testing.T) {
	repo, ctx, cancel, pool := setupRepo(t, NewUserRepository)
	defer cancel()
	err := loadFixtureFile(pool, "./fixtures/user_find_by_id.sql")
	require.NoError(t, err)

	tests := []struct {
		name    string
		id      string
		want    user.User
		wantErr bool
	}{
		{"existing user", "1", user.User{
			ID: "1", LoginHash: "user1hash", PasswordHash: "user1password-hash"}, false},
		{"not found", "100500", user.User{}, true},
		{"bad ID", "not-int", user.User{}, true},
		{"empty ID", "", user.User{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repo.FindByID(ctx, tt.id)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, user.User{}, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
