package apitests

import (
	"net/http"
	"testing"

	"avenue/backend/sdk"
)

func TestLoginRejectsBadPassword(t *testing.T) {
	client, h := authedClient(t)
	_ = h // just needed to force the skip-if-unreachable / login checks above

	meta, err := client.LoginMeta(http.Header{})
	if err != nil {
		t.Fatalf("LoginMeta: %v", err)
	}
	if meta.RegistrationEnabled != "true" {
		t.Skip("skipping: registration disabled, no throwaway account to probe a bad password against")
	}

	email := uniqueName("badlogin") + "@example.com"
	if _, err := client.Register(http.Header{}, sdk.RegisterRequest{
		Email:     email,
		Password:  "CorrectPassw0rd!",
		FirstName: "Bad",
		LastName:  "Login",
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	if _, err := client.Login(http.Header{}, sdk.LoginRequest{Email: email, Password: "WrongPassw0rd!"}); err == nil {
		t.Error("expected login with an incorrect password to fail, got nil error")
	}
}

func TestDashboard(t *testing.T) {
	client, h := authedClient(t)

	dashboard, err := client.DashboardInfo(h)
	if err != nil {
		t.Fatalf("DashboardInfo: %v", err)
	}
	if dashboard.MaxFileSize <= 0 {
		t.Errorf("expected a positive MaxFileSize, got %d", dashboard.MaxFileSize)
	}
}
