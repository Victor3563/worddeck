package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Параметры записываются в каждый хэш, поэтому их можно повысить,
// не ломая уже сохранённые пароли.
type Argon2Params struct {
	Memory  uint32 // КиБ
	Time    uint32 // итерации
	Threads uint8
}

// Значения из OWASP Password Storage Cheat Sheet.
var DefaultArgon2 = Argon2Params{Memory: 19 * 1024, Time: 2, Threads: 1}

const (
	saltLen = 16
	keyLen  = 32
)

var errMalformedHash = errors.New("malformed password hash")

// Формат PHC: $argon2id$v=19$m=...,t=...,p=...$salt$hash
func HashPassword(password string, p Argon2Params) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, keyLen)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Threads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func CheckPassword(password, encoded string) (bool, error) {
	p, salt, want, err := parseHash(encoded)
	if err != nil {
		return false, err
	}
	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

func parseHash(encoded string) (p Argon2Params, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return p, nil, nil, errMalformedHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return p, nil, nil, errMalformedHash
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil {
		return p, nil, nil, errMalformedHash
	}
	enc := base64.RawStdEncoding
	if salt, err = enc.DecodeString(parts[4]); err != nil {
		return p, nil, nil, errMalformedHash
	}
	if key, err = enc.DecodeString(parts[5]); err != nil || len(key) == 0 {
		return p, nil, nil, errMalformedHash
	}
	return p, salt, key, nil
}
