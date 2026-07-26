package handlers

import "testing"

func TestExtractSessionToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		queryToken string
		wantToken  string
		wantOK     bool
	}{
		{
			name:       "valid authorization header",
			authHeader: "Token abc123",
			queryToken: "",
			wantToken:  "abc123",
			wantOK:     true,
		},
		{
			name:       "falls back to query token when header empty",
			authHeader: "",
			queryToken: "xyz789",
			wantToken:  "xyz789",
			wantOK:     true,
		},
		{
			name:       "header takes precedence over query token",
			authHeader: "Token abc123",
			queryToken: "xyz789",
			wantToken:  "abc123",
			wantOK:     true,
		},
		{
			name:       "no header and no query token",
			authHeader: "",
			queryToken: "",
			wantToken:  "",
			wantOK:     false,
		},
		{
			name:       "malformed header missing Token prefix",
			authHeader: "Bearer abc123",
			queryToken: "",
			wantToken:  "",
			wantOK:     false,
		},
		{
			name:       "header with only the Token prefix and no token",
			authHeader: "Token ",
			queryToken: "",
			wantToken:  "",
			wantOK:     true,
		},
		{
			name:       "header containing the literal 'Token ' twice",
			authHeader: "Token Token abc123",
			queryToken: "",
			wantToken:  "",
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, ok := extractSessionToken(tt.authHeader, tt.queryToken)
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if token != tt.wantToken {
				t.Errorf("token = %q, want %q", token, tt.wantToken)
			}
		})
	}
}
