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

// dummyHash is a fixed bcrypt hash, computed once at package load, used to equalise login timing on
// the user-not-found path (see DummyPasswordCheck).
var dummyHash = func() []byte {
	h, err := bcrypt.GenerateFromPassword([]byte("1trade-login-timing-equalizer"), bcrypt.DefaultCost)
	if err != nil { // bcrypt only errors on an absurd cost; never at DefaultCost.
		panic(err)
	}
	return h
}()

// DummyPasswordCheck runs a bcrypt comparison against a fixed hash and always returns false. Call it
// on login's user-not-found path so that path costs the same as verifying a real password —
// otherwise response latency reveals which emails are registered (account enumeration).
func DummyPasswordCheck(plaintext string) bool {
	return bcrypt.CompareHashAndPassword(dummyHash, []byte(plaintext)) == nil
}
