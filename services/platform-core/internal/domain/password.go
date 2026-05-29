package domain

import "golang.org/x/crypto/bcrypt"

// HashPassword returns a bcrypt hash of the plaintext password, safe to store. bcrypt embeds its own
// random salt and a tunable cost; we use the library default cost.
func HashPassword(plaintext string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// VerifyPassword reports whether plaintext matches the stored bcrypt hash. The comparison is
// constant-time within bcrypt, so it doesn't leak timing about how much of the hash matched.
func VerifyPassword(hash, plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext)) == nil
}
