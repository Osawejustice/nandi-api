package utils

import "testing"

func TestPasswordHashRoundTrip(t *testing.T) {
	hash, err := HashPassword("changeme1")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "changeme1" {
		t.Fatal("hash must not equal plaintext")
	}
	if !CheckPassword(hash, "changeme1") {
		t.Fatal("expected match")
	}
	if CheckPassword(hash, "wrong") {
		t.Fatal("expected mismatch")
	}
}
