package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const sessionTokenBytes = 32

// В БД хранится только хэш: по утёкшей базе нельзя войти от чужого имени.
func NewSessionToken() (token string, hash []byte, err error) {
	token, err = RandomToken(sessionTokenBytes)
	if err != nil {
		return "", nil, err
	}
	return token, HashToken(token), nil
}

func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

func RandomToken(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
