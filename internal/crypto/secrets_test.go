package crypto

import (
	"crypto/rand"
	"testing"
)

func TestNewSecretManager(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	sm, err := NewSecretManager(key)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sm == nil {
		t.Fatal("expected non-nil secret manager")
	}
}

func TestNewSecretManagerInvalidKeySize(t *testing.T) {
	tests := []int{16, 24, 31, 33}

	for _, size := range tests {
		key := make([]byte, size)
		rand.Read(key)

		_, err := NewSecretManager(key)
		if err == nil {
			t.Fatalf("expected error for key size %d", size)
		}
	}
}

func TestEncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	sm, err := NewSecretManager(key)
	if err != nil {
		t.Fatalf("failed to create secret manager: %v", err)
	}

	plaintext := "sk-test-1234567890abcdef"

	ciphertext, err := sm.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	decrypted, err := sm.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted text does not match original: expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptionRandomness(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	sm, err := NewSecretManager(key)
	if err != nil {
		t.Fatalf("failed to create secret manager: %v", err)
	}

	plaintext := "same plaintext"

	ciphertext1, err := sm.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt first time: %v", err)
	}

	ciphertext2, err := sm.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt second time: %v", err)
	}

	if ciphertext1 == ciphertext2 {
		t.Fatal("encryption should produce different ciphertexts for same plaintext due to random nonce")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	sm, err := NewSecretManager(key)
	if err != nil {
		t.Fatalf("failed to create secret manager: %v", err)
	}

	_, err = sm.Decrypt("not-valid-base64!!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptTampered(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)

	sm, err := NewSecretManager(key)
	if err != nil {
		t.Fatalf("failed to create secret manager: %v", err)
	}

	plaintext := "sensitive data"

	ciphertext, err := sm.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("failed to encrypt: %v", err)
	}

	tamperedCiphertext := ciphertext[:len(ciphertext)-2]

	_, err = sm.Decrypt(tamperedCiphertext)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}
