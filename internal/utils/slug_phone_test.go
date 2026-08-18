package utils

import "testing"

func TestSlugify(t *testing.T) {
	if got := Slugify("Acme Ltd!"); got != "acme-ltd" {
		t.Fatalf("got %q", got)
	}
	if got := Slugify("   "); got != "org" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizePhone(t *testing.T) {
	if got := NormalizePhone("+254 700-000 001"); got != "+254700000001" {
		t.Fatalf("got %q", got)
	}
}

func TestPreview(t *testing.T) {
	if got := Preview("hello", 10); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if got := Preview("abcdefghijklmnop", 5); got != "abcde…" {
		t.Fatalf("got %q", got)
	}
}
