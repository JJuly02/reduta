package auth

import "testing"

func TestHashVerifyRoundTrip(t *testing.T) {
	const pw = "correct horse battery staple"
	h, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	ok, err := VerifyPassword(pw, h)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("expected password to verify")
	}
	bad, err := VerifyPassword("wrong", h)
	if err != nil {
		t.Fatalf("verify wrong: %v", err)
	}
	if bad {
		t.Fatal("expected wrong password to fail")
	}
}

func TestVerifyRejectsGarbage(t *testing.T) {
	if _, err := VerifyPassword("x", "not-a-hash"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}
