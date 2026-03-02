package auth

import (
	"encoding/base64"
	"testing"
)

const testEncryptionKey = "my-secret-encryption-key"

func TestEncryptDecryptRoundtrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
		key       string
	}{
		{"simple text", "hello, world!", testEncryptionKey},
		{"empty plaintext", "", testEncryptionKey},
		{"unicode text", "kubevision: Kubernetes 仪表板 🚀", testEncryptionKey},
		{"long text", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.", testEncryptionKey},
		{"short key", "data", "k"},
		{"long key", "data", "a-very-long-key-that-exceeds-thirty-two-bytes-in-length-for-sure"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ciphertext, err := Encrypt(tc.plaintext, tc.key)
			if err != nil {
				t.Fatalf("Encrypt(%q, key) error: %v", tc.plaintext, err)
			}
			if ciphertext == "" {
				t.Fatal("Encrypt returned empty ciphertext")
			}

			// The ciphertext must be valid base64.
			if _, err := base64.StdEncoding.DecodeString(ciphertext); err != nil {
				t.Fatalf("ciphertext is not valid base64: %v", err)
			}

			decrypted, err := Decrypt(ciphertext, tc.key)
			if err != nil {
				t.Fatalf("Decrypt error: %v", err)
			}
			if decrypted != tc.plaintext {
				t.Errorf("Decrypt = %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	plaintext := "sensitive data"
	ciphertext, err := Encrypt(plaintext, testEncryptionKey)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	wrongKeys := []struct {
		name string
		key  string
	}{
		{"completely different key", "wrong-key"},
		{"empty key", ""},
		{"similar key", testEncryptionKey + "x"},
	}

	for _, tc := range wrongKeys {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decrypt(ciphertext, tc.key)
			if err == nil {
				t.Error("Decrypt with wrong key expected error, got nil")
			}
		})
	}
}

func TestEncryptProducesDifferentCiphertexts(t *testing.T) {
	plaintext := "same plaintext every time"
	key := testEncryptionKey

	ct1, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("first Encrypt error: %v", err)
	}
	ct2, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("second Encrypt error: %v", err)
	}

	if ct1 == ct2 {
		t.Error("two Encrypt calls with identical inputs produced identical ciphertexts; expected different nonces")
	}

	// Both must still decrypt to the original plaintext.
	for i, ct := range []string{ct1, ct2} {
		dec, err := Decrypt(ct, key)
		if err != nil {
			t.Fatalf("Decrypt of ciphertext %d error: %v", i+1, err)
		}
		if dec != plaintext {
			t.Errorf("Decrypt of ciphertext %d = %q, want %q", i+1, dec, plaintext)
		}
	}
}

func TestDecryptWithCorruptedCiphertext(t *testing.T) {
	plaintext := "do not tamper"
	ciphertext, err := Encrypt(plaintext, testEncryptionKey)
	if err != nil {
		t.Fatalf("Encrypt error: %v", err)
	}

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"not base64 at all", "!!!not-base64!!!"},
		{"empty string", ""},
		{"truncated ciphertext", ciphertext[:len(ciphertext)/2]},
		{"flipped byte in ciphertext", flipByteInBase64(t, ciphertext)},
		{"too short after decode", base64.StdEncoding.EncodeToString([]byte("short"))},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decrypt(tc.ciphertext, testEncryptionKey)
			if err == nil {
				t.Errorf("Decrypt(%q) expected error, got nil", tc.name)
			}
		})
	}
}

// flipByteInBase64 decodes the base64 ciphertext, flips a byte in the
// encrypted payload (past the nonce), re-encodes, and returns the corrupted
// base64 string.
func flipByteInBase64(t *testing.T, ct string) string {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString(ct)
	if err != nil {
		t.Fatalf("flipByteInBase64: decode error: %v", err)
	}
	// GCM nonce is 12 bytes; flip a byte well past that.
	if len(data) < 14 {
		t.Fatal("flipByteInBase64: ciphertext too short to corrupt")
	}
	data[len(data)-1] ^= 0xff
	return base64.StdEncoding.EncodeToString(data)
}
