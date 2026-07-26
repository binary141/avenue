// Package apitests contains black-box integration tests for the Avenue HTTP
// API. They exercise a real, running server (and real database) over the
// network using the sdk.Client — the same client the CLI and any future
// tooling would use — rather than calling handlers/db functions directly.
//
// These tests need a live server to talk to. Bring one up with:
//
//	docker-compose up -d
//
// then run:
//
//	go test ./api-tests/... -v
//
// If no server is reachable at the configured base URL, every test skips
// itself (rather than failing) so `go test ./...` stays safe to run without
// one. See README.md in this directory for the full list of env vars.
package apitests

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	"avenue/backend/sdk"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func baseURL() string {
	return getenv("AVENUE_TEST_BASE_URL", "http://localhost:8080")
}

// uniqueName returns a name unlikely to collide with anything already in the
// account being tested against (root/shared test accounts may accumulate
// leftovers across runs), so assertions can look for an exact name match
// without tripping over unrelated data.
func uniqueName(prefix string) string {
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), rand.Intn(1_000_000))
}

// testClient pings the configured server and skips the calling test if it's
// unreachable, so these tests don't fail (or hang) in environments without
// docker-compose running.
func testClient(t *testing.T) *sdk.Client {
	t.Helper()

	client := sdk.NewClient(baseURL())
	if _, err := client.Ping(http.Header{}); err != nil {
		t.Skipf("skipping: could not reach Avenue server at %s: %v", baseURL(), err)
	}
	return client
}

// authedClient returns a client plus the Authorization header for a logged
// in user, along with a t.Cleanup that logs the session out. It registers a
// fresh, randomly-named user for the run when the server has self
// registration enabled (as docker-compose's dev config does); otherwise it
// falls back to logging in with AVENUE_TEST_EMAIL / AVENUE_TEST_PASSWORD
// (defaulting to the root user seeded by ROOT_USER_EMAIL/ROOT_USER_PASSWORD).
func authedClient(t *testing.T) (*sdk.Client, http.Header) {
	t.Helper()

	client := testClient(t)

	meta, err := client.LoginMeta(http.Header{})
	if err != nil {
		t.Fatalf("LoginMeta: %v", err)
	}

	email := getenv("AVENUE_TEST_EMAIL", "")
	password := getenv("AVENUE_TEST_PASSWORD", "")

	if meta.RegistrationEnabled == "true" && email == "" {
		email = uniqueName("apitest") + "@example.com"
		password = "Sup3rSecretPassw0rd!"

		if _, err := client.Register(http.Header{}, sdk.RegisterRequest{
			Email:     email,
			Password:  password,
			FirstName: "API",
			LastName:  "Test",
		}); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}

	if email == "" {
		email = "root@gmail.com"
	}
	if password == "" {
		password = "password"
	}

	login, err := client.Login(http.Header{}, sdk.LoginRequest{Email: email, Password: password})
	if err != nil {
		t.Skipf("skipping: could not log in as %s (set AVENUE_TEST_EMAIL/AVENUE_TEST_PASSWORD to match a real account): %v", email, err)
	}

	h := http.Header{"Authorization": []string{"Token " + login.SessionID}}

	t.Cleanup(func() {
		_, _ = client.Logout(h)
	})

	return client, h
}
