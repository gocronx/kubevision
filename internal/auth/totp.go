package auth

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	totpIssuer        = "KubeVision"
	recoveryCodeLen   = 8
	recoveryCodeChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // unambiguous chars
)

// GenerateSecret creates a new TOTP secret for the given username.
// It returns the raw base32 secret and the otpauth:// URL for QR code generation.
func GenerateSecret(username string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: username,
		Period:      30,
		SecretSize:  20,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", "", fmt.Errorf("generate totp key: %w", err)
	}
	return key.Secret(), key.URL(), nil
}

// ValidateCode verifies a 6-digit TOTP code against the given base32 secret.
// It accepts codes from the current and adjacent 30-second windows (skew=1).
func ValidateCode(secret, code string) bool {
	return totp.Validate(code, secret)
}

// ValidateCodeWithOptions verifies a TOTP code with explicit options.
// Allows a ±1 period skew to handle slight clock drift.
func ValidateCodeWithOptions(secret, code string) bool {
	// Use a slightly wider validation window (skew of 1 = ±30 seconds)
	valid, err := totp.ValidateCustom(code, secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	if err != nil {
		return false
	}
	return valid
}

// GenerateRecoveryCodes creates count random recovery codes.
// Each code is recoveryCodeLen characters from an unambiguous alphanumeric set,
// formatted as XXXX-XXXX for readability.
func GenerateRecoveryCodes(count int) ([]string, error) {
	codes := make([]string, 0, count)
	charset := []byte(recoveryCodeChars)
	for i := 0; i < count; i++ {
		buf := make([]byte, recoveryCodeLen)
		for j := range buf {
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
			if err != nil {
				return nil, fmt.Errorf("generate recovery code: %w", err)
			}
			buf[j] = charset[idx.Int64()]
		}
		// Format as XXXX-XXXX
		code := string(buf[:4]) + "-" + string(buf[4:])
		codes = append(codes, code)
	}
	return codes, nil
}

// EncryptSecret encrypts a plaintext TOTP secret using AES-256-GCM.
// It delegates to the shared Encrypt function in the auth package.
func EncryptSecret(plaintext, key string) (string, error) {
	return Encrypt(plaintext, key)
}

// DecryptSecret decrypts an AES-256-GCM-encrypted TOTP secret.
// It delegates to the shared Decrypt function in the auth package.
func DecryptSecret(ciphertext, key string) (string, error) {
	return Decrypt(ciphertext, key)
}

// OTPSecretFromBase32 normalises a raw secret string into standard base32 encoding.
// The pquerna/otp library stores secrets in base32 without padding; this helper
// ensures the secret is always stored in a consistent format.
func OTPSecretFromBase32(raw string) (string, error) {
	// Decode then re-encode to normalise.
	raw = strings.ToUpper(strings.TrimSpace(raw))
	// Add padding if needed for decoding.
	padded := raw
	if mod := len(padded) % 8; mod != 0 {
		padded += strings.Repeat("=", 8-mod)
	}
	decoded, err := base32.StdEncoding.DecodeString(padded)
	if err != nil {
		return "", fmt.Errorf("invalid base32 secret: %w", err)
	}
	// Re-encode without padding (pquerna/otp convention).
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(decoded), nil
}
