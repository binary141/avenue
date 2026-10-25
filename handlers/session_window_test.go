package handlers

import (
	"testing"
	"time"

	"avenue/backend/db"
)

func TestSessionWindow(t *testing.T) {
	tests := []struct {
		clientType string
		want       time.Duration
	}{
		{clientType: db.ClientTypeWeb, want: 15 * time.Minute},
		{clientType: db.ClientTypeMobile, want: 90 * 24 * time.Hour},
		// Anything unexpected gets the web window, the shorter of the two.
		{clientType: "", want: 15 * time.Minute},
	}

	for _, tt := range tests {
		if got := sessionWindow(tt.clientType); got != tt.want {
			t.Errorf("sessionWindow(%q) = %v, want %v", tt.clientType, got, tt.want)
		}
	}
}
