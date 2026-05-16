package auth

import "golang.org/x/crypto/bcrypt"

const bcryptCost = 12

// HashPassword hashes a plaintext password with bcrypt cost 12.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func checkPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
