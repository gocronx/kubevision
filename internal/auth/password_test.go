package auth

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("returns a valid bcrypt hash", func(t *testing.T) {
		hash, err := HashPassword("mysecretpassword")
		if err != nil {
			t.Fatalf("HashPassword returned error: %v", err)
		}
		if hash == "" {
			t.Fatal("HashPassword returned empty hash")
		}
		// A valid bcrypt hash can be parsed by bcrypt.Cost without error.
		cost, err := bcrypt.Cost([]byte(hash))
		if err != nil {
			t.Fatalf("hash is not valid bcrypt: %v", err)
		}
		if cost != bcryptCost {
			t.Errorf("expected cost %d, got %d", bcryptCost, cost)
		}
	})

	t.Run("empty string password", func(t *testing.T) {
		hash, err := HashPassword("")
		if err != nil {
			t.Fatalf("HashPassword with empty string returned error: %v", err)
		}
		if hash == "" {
			t.Fatal("HashPassword with empty string returned empty hash")
		}
		// The empty-string hash should still be valid bcrypt.
		if _, err := bcrypt.Cost([]byte(hash)); err != nil {
			t.Fatalf("hash of empty string is not valid bcrypt: %v", err)
		}
	})

	t.Run("different calls produce different hashes due to random salt", func(t *testing.T) {
		hash1, err := HashPassword("samepassword")
		if err != nil {
			t.Fatalf("first HashPassword call returned error: %v", err)
		}
		hash2, err := HashPassword("samepassword")
		if err != nil {
			t.Fatalf("second HashPassword call returned error: %v", err)
		}
		if hash1 == hash2 {
			t.Error("two HashPassword calls with the same input produced identical hashes; expected different salts")
		}
	})
}

func TestCheckPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		input    string
		want     bool
	}{
		{
			name:     "correct password returns true",
			password: "correcthorsebatterystaple",
			input:    "correcthorsebatterystaple",
			want:     true,
		},
		{
			name:     "wrong password returns false",
			password: "correcthorsebatterystaple",
			input:    "wrongpassword",
			want:     false,
		},
		{
			name:     "empty password matches empty hash input",
			password: "",
			input:    "",
			want:     true,
		},
		{
			name:     "empty input against non-empty password returns false",
			password: "notempty",
			input:    "",
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := HashPassword(tc.password)
			if err != nil {
				t.Fatalf("HashPassword(%q) returned error: %v", tc.password, err)
			}
			got := CheckPassword(tc.input, hash)
			if got != tc.want {
				t.Errorf("CheckPassword(%q, hash(%q)) = %v, want %v", tc.input, tc.password, got, tc.want)
			}
		})
	}
}
