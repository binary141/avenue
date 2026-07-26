package db

import (
	"strings"
	"testing"
)

func TestGenerateTokenLengthAndCharset(t *testing.T) {
	for i := 0; i < 100; i++ {
		token, err := generateToken()
		if err != nil {
			t.Fatalf("generateToken() error = %v", err)
		}
		if len(token) != 32 {
			t.Fatalf("generateToken() len = %d, want 32", len(token))
		}
		for _, c := range token {
			if !strings.ContainsRune(charset, c) {
				t.Fatalf("generateToken() produced out-of-charset rune %q in %q", c, token)
			}
		}
	}
}

func TestGenerateTokenIsRandom(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		token, err := generateToken()
		if err != nil {
			t.Fatalf("generateToken() error = %v", err)
		}
		if seen[token] {
			t.Fatalf("generateToken() produced a duplicate token: %q", token)
		}
		seen[token] = true
	}
}
