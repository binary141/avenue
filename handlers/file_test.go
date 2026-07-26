package handlers

import "testing"

func TestComputeUploadLimit(t *testing.T) {
	tests := []struct {
		name          string
		quota         int64
		used          int64
		envMax        int64
		wantLimit     int64
		wantOverQuota bool
	}{
		{
			name:          "unlimited quota uses env max",
			quota:         0,
			used:          1_000_000,
			envMax:        500,
			wantLimit:     500,
			wantOverQuota: false,
		},
		{
			name:          "used equal to quota is over quota",
			quota:         100,
			used:          100,
			envMax:        500,
			wantLimit:     0,
			wantOverQuota: true,
		},
		{
			name:          "used beyond quota is over quota",
			quota:         100,
			used:          150,
			envMax:        500,
			wantLimit:     0,
			wantOverQuota: true,
		},
		{
			name:          "remaining quota smaller than env max is clamped",
			quota:         100,
			used:          80,
			envMax:        500,
			wantLimit:     20,
			wantOverQuota: false,
		},
		{
			name:          "env max smaller than remaining quota wins",
			quota:         1000,
			used:          10,
			envMax:        500,
			wantLimit:     500,
			wantOverQuota: false,
		},
		{
			name:          "remaining quota exactly equal to env max",
			quota:         600,
			used:          100,
			envMax:        500,
			wantLimit:     500,
			wantOverQuota: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, overQuota := computeUploadLimit(tt.quota, tt.used, tt.envMax)
			if overQuota != tt.wantOverQuota {
				t.Errorf("overQuota = %v, want %v", overQuota, tt.wantOverQuota)
			}
			if limit != tt.wantLimit {
				t.Errorf("limit = %d, want %d", limit, tt.wantLimit)
			}
		})
	}
}
