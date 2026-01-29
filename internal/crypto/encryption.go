// Package crypto provides encryption utilities for secure data storage.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrInvalidKey indicates the encryption key is invalid.
	ErrInvalidKey = errors.New("encryption key must be 32 bytes for AES-256")
	// ErrInvalidCiphertext indicates the ciphertext is invalid or corrupted.
	ErrInvalidCiphertext = errors.New("invalid or corrupted ciphertext")
	// ErrDecryptionFailed indicates decryption failed.
	ErrDecryptionFailed = errors.New("decryption failed")
)

// Encryptor provides AES-256-GCM encryption and decryption.
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new Encryptor with the given key.
// The key must be exactly 32 bytes for AES-256.
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) != 32 {
		return nil, ErrInvalidKey
	}
	return &Encryptor{key: key}, nil
}

// NewEncryptorFromString creates a new Encryptor from a base64-encoded key.
func NewEncryptorFromString(keyBase64 string) (*Encryptor, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 key: %w", err)
	}
	return NewEncryptor(key)
}

// GenerateKey generates a new random 32-byte encryption key.
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyBase64 generates a new random encryption key and returns it as base64.
func GenerateKeyBase64() (string, error) {
	key, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Encrypt encrypts plaintext using AES-256-GCM.
// Returns the ciphertext with nonce prepended.
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce to ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext that was encrypted with Encrypt.
// Expects the nonce to be prepended to the ciphertext.
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrInvalidCiphertext
	}

	// Extract nonce and actual ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext.
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts base64-encoded ciphertext and returns the plaintext string.
func (e *Encryptor) DecryptString(ciphertextBase64 string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("invalid base64 ciphertext: %w", err)
	}
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptToBytes encrypts a string and returns raw ciphertext bytes.
// Useful for storing in BYTEA database columns.
func (e *Encryptor) EncryptToBytes(plaintext string) ([]byte, error) {
	return e.Encrypt([]byte(plaintext))
}

// DecryptFromBytes decrypts raw ciphertext bytes and returns the plaintext string.
// Useful for retrieving from BYTEA database columns.
func (e *Encryptor) DecryptFromBytes(ciphertext []byte) (string, error) {
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
