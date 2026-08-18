package utils

import (
	"strings"
	"unicode"
)

// NormalizePhone strips spaces, dashes, and parentheses. Keeps a leading +.
func NormalizePhone(raw string) string {
	raw = strings.TrimSpace(raw)
	var b strings.Builder
	for i, r := range raw {
		if r == '+' && i == 0 {
			b.WriteRune(r)
			continue
		}
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func Preview(body string, n int) string {
	body = strings.TrimSpace(body)
	if n <= 0 {
		n = 140
	}
	runes := []rune(body)
	if len(runes) <= n {
		return body
	}
	return string(runes[:n]) + "…"
}
