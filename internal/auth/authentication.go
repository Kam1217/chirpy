package auth

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bcrypt.GenerateFromPassword()
	return "", nil
}
