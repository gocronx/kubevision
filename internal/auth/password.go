package auth

import "golang.org/x/crypto/bcrypt"

// bcryptCost is the work factor used when hashing passwords. Cost 12 strikes a
// good balance between brute-force resistance and acceptable login latency on
// modern hardware (~250 ms per hash).
const bcryptCost = 12

// HashPassword returns the bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
// It returns true if the password matches the hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
