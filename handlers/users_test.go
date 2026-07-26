package handlers

import (
	"testing"

	"avenue/backend/sdk"
)

func ptr[T any](v T) *T { return &v }

func TestValidateProfileUpdate(t *testing.T) {
	tests := []struct {
		name          string
		callerID      string
		callerIsAdmin bool
		req           sdk.UpdateProfileRequest
		emailConflict bool
		hasOtherAdmin bool
		wantErr       string
	}{
		{
			name:          "self edit is allowed",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 1, FirstName: ptr("Ada")},
			wantErr:       "",
		},
		{
			name:          "non-admin editing another user is rejected",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 2, FirstName: ptr("Ada")},
			wantErr:       "only admin users can edit another user's information",
		},
		{
			name:          "admin editing another user is allowed",
			callerID:      "1",
			callerIsAdmin: true,
			req:           sdk.UpdateProfileRequest{ID: 2, FirstName: ptr("Ada")},
			wantErr:       "",
		},
		{
			name:          "email conflict is rejected",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 1, Email: ptr("taken@example.com")},
			emailConflict: true,
			wantErr:       "email already exists",
		},
		{
			name:          "self password change without current password is rejected",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 1, Password: ptr("newpassword123")},
			wantErr:       "current password is required",
		},
		{
			name:          "self password change with empty current password is rejected",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 1, Password: ptr("newpassword123"), CurrentPassword: ptr("")},
			wantErr:       "current password is required",
		},
		{
			name:          "self password change with current password supplied passes validation",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 1, Password: ptr("newpassword123"), CurrentPassword: ptr("oldpass")},
			wantErr:       "",
		},
		{
			name:          "admin resetting another user's password doesn't need current password",
			callerID:      "1",
			callerIsAdmin: true,
			req:           sdk.UpdateProfileRequest{ID: 2, Password: ptr("newpassword123")},
			wantErr:       "",
		},
		{
			name:          "demoting the last admin is rejected",
			callerID:      "1",
			callerIsAdmin: true,
			req:           sdk.UpdateProfileRequest{ID: 1, IsAdmin: ptr(false)},
			hasOtherAdmin: false,
			wantErr:       "application requires at least one admin user",
		},
		{
			name:          "demoting an admin when others exist is allowed",
			callerID:      "1",
			callerIsAdmin: true,
			req:           sdk.UpdateProfileRequest{ID: 1, IsAdmin: ptr(false)},
			hasOtherAdmin: true,
			wantErr:       "",
		},
		{
			name:          "promoting to admin is allowed regardless of other admins",
			callerID:      "1",
			callerIsAdmin: true,
			req:           sdk.UpdateProfileRequest{ID: 2, IsAdmin: ptr(true)},
			hasOtherAdmin: false,
			wantErr:       "",
		},
		{
			name:          "non-admin caller can't change IsAdmin (no-op, not rejected here)",
			callerID:      "1",
			callerIsAdmin: false,
			req:           sdk.UpdateProfileRequest{ID: 1, IsAdmin: ptr(false)},
			hasOtherAdmin: false,
			wantErr:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProfileUpdate(tt.callerID, tt.callerIsAdmin, tt.req, tt.emailConflict, tt.hasOtherAdmin)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("err = %q, want nil", err.Error())
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("err = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
