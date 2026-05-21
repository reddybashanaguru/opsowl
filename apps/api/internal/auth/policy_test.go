package auth

import (
	"testing"

	db "github.com/reddybashanaguru/opsowl/internal/db/generated"
)

func TestPasswordPolicy_Validate(t *testing.T) {
	policy := PasswordPolicy{
		MinLength:     8,
		RequireUpper:  true,
		RequireNumber: true,
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "Secure@123", false},
		{"too short", "Ab1", true},
		{"no uppercase", "lowercase123", true},
		{"no number", "NoNumbers!", true},
		{"exactly min length", "Abcdef1!", false},
		{"long valid password", "VeryLongAndSecure123!@#", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.Validate(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%q) error = %v, wantErr %v", tt.password, err, tt.wantErr)
			}
		})
	}
}

func TestPasswordPolicy_NoRequirements(t *testing.T) {
	policy := PasswordPolicy{MinLength: 4}
	if err := policy.Validate("abcd"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestIsPasswordExpired_NeverExpires(t *testing.T) {
	user := db.User{}
	if IsPasswordExpired(user, 0) {
		t.Error("expected not expired when expiryDays=0")
	}
}
