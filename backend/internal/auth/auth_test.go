package auth

import (
	"strings"
	"testing"
)

var cheap = Argon2Params{Memory: 64, Time: 1, Threads: 1}

func TestPasswordRoundTrip(t *testing.T) {
	h, err := HashPassword("correct horse", cheap)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=64,t=1,p=1$") {
		t.Fatalf("unexpected hash format: %s", h)
	}
	if ok, err := CheckPassword("correct horse", h); err != nil || !ok {
		t.Fatalf("valid password rejected: ok=%v err=%v", ok, err)
	}
	if ok, _ := CheckPassword("wrong horse", h); ok {
		t.Fatal("wrong password accepted")
	}
}

func TestHashesAreSalted(t *testing.T) {
	a, _ := HashPassword("same", cheap)
	b, _ := HashPassword("same", cheap)
	if a == b {
		t.Fatal("two hashes of the same password are equal")
	}
}

func TestCheckPasswordRejectsMalformed(t *testing.T) {
	for _, h := range []string{"", "plain", "$argon2i$v=19$m=64,t=1,p=1$c2FsdA$a2V5", "$argon2id$v=1$m=64,t=1,p=1$c2FsdA$a2V5", "$argon2id$v=19$m=64,t=1,p=1$!!$a2V5"} {
		if ok, err := CheckPassword("x", h); ok || err == nil {
			t.Errorf("hash %q: ok=%v err=%v, want rejection", h, ok, err)
		}
	}
}

func TestSessionToken(t *testing.T) {
	tok, hash, err := NewSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) < 40 {
		t.Fatalf("token too short: %q", tok)
	}
	if string(HashToken(tok)) != string(hash) {
		t.Fatal("HashToken does not match the hash returned with the token")
	}
	tok2, _, _ := NewSessionToken()
	if tok == tok2 {
		t.Fatal("tokens are not random")
	}
}
