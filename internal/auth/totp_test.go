package auth

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
)

// ---------------------------------------------------------------------------
// GenerateSecret tests
// ---------------------------------------------------------------------------

func TestGenerateSecret_ReturnsNonEmptyFields(t *testing.T) {
	secret, otpauthURL, err := GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret returned error: %v", err)
	}
	if secret == "" {
		t.Error("GenerateSecret returned empty secret")
	}
	if otpauthURL == "" {
		t.Error("GenerateSecret returned empty otpauthURL")
	}
}

func TestGenerateSecret_URLContainsIssuerAndAccount(t *testing.T) {
	username := "alice"
	_, otpauthURL, err := GenerateSecret(username)
	if err != nil {
		t.Fatalf("GenerateSecret returned error: %v", err)
	}

	if !strings.Contains(otpauthURL, "KubeVision") {
		t.Errorf("otpauthURL %q does not contain issuer %q", otpauthURL, "KubeVision")
	}
	if !strings.Contains(otpauthURL, username) {
		t.Errorf("otpauthURL %q does not contain account name %q", otpauthURL, username)
	}
	if !strings.HasPrefix(otpauthURL, "otpauth://totp/") {
		t.Errorf("otpauthURL %q does not start with otpauth://totp/", otpauthURL)
	}
}

func TestGenerateSecret_SecretIsValidBase32(t *testing.T) {
	secret, _, err := GenerateSecret("bob")
	if err != nil {
		t.Fatalf("GenerateSecret returned error: %v", err)
	}

	// The secret should be decodable as base32 (with or without padding).
	padded := secret
	if mod := len(padded) % 8; mod != 0 {
		padded += strings.Repeat("=", 8-mod)
	}
	if _, err := base32.StdEncoding.DecodeString(padded); err != nil {
		t.Errorf("secret %q is not valid base32: %v", secret, err)
	}
}

func TestGenerateSecret_DifferentCallsProduceDifferentSecrets(t *testing.T) {
	secret1, _, err := GenerateSecret("user1")
	if err != nil {
		t.Fatalf("first GenerateSecret error: %v", err)
	}
	secret2, _, err := GenerateSecret("user2")
	if err != nil {
		t.Fatalf("second GenerateSecret error: %v", err)
	}
	if secret1 == secret2 {
		t.Error("two GenerateSecret calls produced identical secrets; expected randomness")
	}
}

// ---------------------------------------------------------------------------
// ValidateCode tests
// ---------------------------------------------------------------------------

func TestValidateCode_ValidCodeReturnsTrue(t *testing.T) {
	secret, _, err := GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	// Generate the current valid code using the same library.
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("totp.GenerateCode error: %v", err)
	}

	if !ValidateCode(secret, code) {
		t.Errorf("ValidateCode with a freshly generated code returned false")
	}
}

func TestValidateCode_InvalidCodeReturnsFalse(t *testing.T) {
	secret, _, err := GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	tests := []struct {
		name string
		code string
	}{
		{"all zeros", "000000"},
		{"too short", "12345"},
		{"too long", "1234567"},
		{"non-numeric", "abcdef"},
		{"empty", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if ValidateCode(secret, tc.code) {
				t.Errorf("ValidateCode with code %q expected false, got true", tc.code)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ValidateCodeWithOptions tests
// ---------------------------------------------------------------------------

func TestValidateCodeWithOptions_ValidCodeReturnsTrue(t *testing.T) {
	secret, _, err := GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("totp.GenerateCode error: %v", err)
	}

	if !ValidateCodeWithOptions(secret, code) {
		t.Errorf("ValidateCodeWithOptions with a freshly generated code returned false")
	}
}

func TestValidateCodeWithOptions_InvalidCodeReturnsFalse(t *testing.T) {
	secret, _, err := GenerateSecret("testuser")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	invalidCodes := []struct {
		name string
		code string
	}{
		{"all nines", "999999"},
		{"non-numeric", "abcdef"},
		{"empty", ""},
	}

	for _, tc := range invalidCodes {
		t.Run(tc.name, func(t *testing.T) {
			if ValidateCodeWithOptions(secret, tc.code) {
				t.Errorf("ValidateCodeWithOptions(%q) expected false, got true", tc.code)
			}
		})
	}
}

func TestValidateCodeWithOptions_WrongSecretReturnsFalse(t *testing.T) {
	secret, _, err := GenerateSecret("user-a")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	otherSecret, _, err := GenerateSecret("user-b")
	if err != nil {
		t.Fatalf("second GenerateSecret error: %v", err)
	}

	// Generate a valid code for the first secret.
	code, err := totp.GenerateCode(secret, time.Now().UTC())
	if err != nil {
		t.Fatalf("totp.GenerateCode error: %v", err)
	}

	// Validating the first user's code against the second user's secret must fail.
	if ValidateCodeWithOptions(otherSecret, code) {
		t.Error("ValidateCodeWithOptions with wrong secret expected false, got true")
	}
}

// ---------------------------------------------------------------------------
// GenerateRecoveryCodes tests
// ---------------------------------------------------------------------------

func TestGenerateRecoveryCodes_ReturnsRequestedCount(t *testing.T) {
	tests := []struct {
		count int
	}{
		{1}, {5}, {10}, {16},
	}

	for _, tc := range tests {
		codes, err := GenerateRecoveryCodes(tc.count)
		if err != nil {
			t.Fatalf("GenerateRecoveryCodes(%d) returned error: %v", tc.count, err)
		}
		if len(codes) != tc.count {
			t.Errorf("GenerateRecoveryCodes(%d) returned %d codes, want %d", tc.count, len(codes), tc.count)
		}
	}
}

func TestGenerateRecoveryCodes_FormatIsXXXXDashXXXX(t *testing.T) {
	codes, err := GenerateRecoveryCodes(8)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes returned error: %v", err)
	}

	for _, code := range codes {
		parts := strings.Split(code, "-")
		if len(parts) != 2 {
			t.Errorf("code %q does not match XXXX-XXXX format", code)
			continue
		}
		if len(parts[0]) != 4 || len(parts[1]) != 4 {
			t.Errorf("code %q parts have unexpected lengths: %d, %d", code, len(parts[0]), len(parts[1]))
		}
	}
}

func TestGenerateRecoveryCodes_OnlyUseUnambiguousChars(t *testing.T) {
	codes, err := GenerateRecoveryCodes(20)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes returned error: %v", err)
	}

	allowed := recoveryCodeChars
	for _, code := range codes {
		raw := strings.ReplaceAll(code, "-", "")
		for _, ch := range raw {
			if !strings.ContainsRune(allowed, ch) {
				t.Errorf("code %q contains disallowed character %q", code, ch)
			}
		}
	}
}

func TestGenerateRecoveryCodes_CodesAreUnique(t *testing.T) {
	codes, err := GenerateRecoveryCodes(100)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes returned error: %v", err)
	}

	seen := make(map[string]bool, len(codes))
	for _, code := range codes {
		if seen[code] {
			t.Errorf("duplicate recovery code found: %q", code)
		}
		seen[code] = true
	}
}

func TestGenerateRecoveryCodes_ZeroCount(t *testing.T) {
	codes, err := GenerateRecoveryCodes(0)
	if err != nil {
		t.Fatalf("GenerateRecoveryCodes(0) returned error: %v", err)
	}
	if len(codes) != 0 {
		t.Errorf("GenerateRecoveryCodes(0) returned %d codes, want 0", len(codes))
	}
}

// ---------------------------------------------------------------------------
// EncryptSecret / DecryptSecret tests
// ---------------------------------------------------------------------------

func TestEncryptDecryptSecret_Roundtrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		key       string
	}{
		{"typical totp secret", "JBSWY3DPEHPK3PXP", "encryption-key-123"},
		{"empty secret", "", "some-key"},
		{"long secret", strings.Repeat("ABCDEFGH", 10), "another-key"},
		{"short key", "MYSECRET", "k"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, err := EncryptSecret(tc.plaintext, tc.key)
			if err != nil {
				t.Fatalf("EncryptSecret error: %v", err)
			}
			if ciphertext == "" && tc.plaintext != "" {
				t.Fatal("EncryptSecret returned empty ciphertext for non-empty plaintext")
			}

			decrypted, err := DecryptSecret(ciphertext, tc.key)
			if err != nil {
				t.Fatalf("DecryptSecret error: %v", err)
			}
			if decrypted != tc.plaintext {
				t.Errorf("DecryptSecret = %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestDecryptSecret_WrongKeyFails(t *testing.T) {
	plaintext := "JBSWY3DPEHPK3PXP"
	key := "correct-key"

	ciphertext, err := EncryptSecret(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptSecret error: %v", err)
	}

	wrongKeys := []struct {
		name string
		key  string
	}{
		{"completely different", "wrong-key"},
		{"empty key", ""},
		{"similar key", key + "x"},
	}

	for _, tc := range wrongKeys {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptSecret(ciphertext, tc.key)
			if err == nil {
				t.Error("DecryptSecret with wrong key expected error, got nil")
			}
		})
	}
}

func TestEncryptSecret_DifferentCallsProduceDifferentCiphertexts(t *testing.T) {
	plaintext := "JBSWY3DPEHPK3PXP"
	key := "test-key"

	ct1, err := EncryptSecret(plaintext, key)
	if err != nil {
		t.Fatalf("first EncryptSecret error: %v", err)
	}
	ct2, err := EncryptSecret(plaintext, key)
	if err != nil {
		t.Fatalf("second EncryptSecret error: %v", err)
	}

	// AES-GCM uses a random nonce, so two encryptions of the same plaintext must differ.
	if ct1 == ct2 {
		t.Error("two EncryptSecret calls with identical inputs produced identical ciphertexts; expected random nonce")
	}
}

// ---------------------------------------------------------------------------
// OTPSecretFromBase32 tests
// ---------------------------------------------------------------------------

func TestOTPSecretFromBase32_NormalisesValidSecret(t *testing.T) {
	// Generate a real secret and verify round-trip normalisation.
	secret, _, err := GenerateSecret("normalize-test")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	normalised, err := OTPSecretFromBase32(secret)
	if err != nil {
		t.Fatalf("OTPSecretFromBase32 error: %v", err)
	}
	if normalised == "" {
		t.Error("OTPSecretFromBase32 returned empty string")
	}

	// The normalised secret must also produce valid TOTP codes.
	code, err := totp.GenerateCode(normalised, time.Now().UTC())
	if err != nil {
		t.Fatalf("totp.GenerateCode on normalised secret error: %v", err)
	}
	if !ValidateCode(normalised, code) {
		t.Error("code generated from normalised secret failed validation")
	}
}

func TestOTPSecretFromBase32_LowercaseInputNormalises(t *testing.T) {
	// pquerna/otp returns uppercase base32; verify lowercase is also normalised.
	secret, _, err := GenerateSecret("lowercase-test")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	lowered := strings.ToLower(secret)
	normalised, err := OTPSecretFromBase32(lowered)
	if err != nil {
		t.Fatalf("OTPSecretFromBase32 with lowercase secret error: %v", err)
	}

	// After normalisation the result must still validate codes.
	code, err := totp.GenerateCode(normalised, time.Now().UTC())
	if err != nil {
		t.Fatalf("totp.GenerateCode error: %v", err)
	}
	if !ValidateCode(normalised, code) {
		t.Error("code generated from lowercased+normalised secret failed validation")
	}
}

func TestOTPSecretFromBase32_InvalidBase32ReturnsError(t *testing.T) {
	invalidSecrets := []struct {
		name   string
		secret string
	}{
		{"random garbage", "not-valid-base32-!!!"},
		{"non-alphabet chars", "1234@#$%"},
	}

	for _, tc := range invalidSecrets {
		t.Run(tc.name, func(t *testing.T) {
			_, err := OTPSecretFromBase32(tc.secret)
			if err == nil {
				t.Errorf("OTPSecretFromBase32(%q) expected error, got nil", tc.secret)
			}
		})
	}
}

func TestOTPSecretFromBase32_IdempotentOnAlreadyNormalisedSecret(t *testing.T) {
	secret, _, err := GenerateSecret("idempotent-test")
	if err != nil {
		t.Fatalf("GenerateSecret error: %v", err)
	}

	first, err := OTPSecretFromBase32(secret)
	if err != nil {
		t.Fatalf("first OTPSecretFromBase32 error: %v", err)
	}
	second, err := OTPSecretFromBase32(first)
	if err != nil {
		t.Fatalf("second OTPSecretFromBase32 error: %v", err)
	}
	if first != second {
		t.Errorf("OTPSecretFromBase32 is not idempotent: first=%q second=%q", first, second)
	}
}
