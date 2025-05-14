package user

import "context"

type User struct {
	ID           string `json:"id"`
	LoginHash    string `json:"login_hash"`
	PasswordHash string `json:"password_hash"`
}

type Repository interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id string) (*User, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
}
