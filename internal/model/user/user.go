package user

type User struct {
	ID           string `json:"id"`
	LoginHash    string `json:"login_hash"`
	PasswordHash string `json:"password_hash"`
}
