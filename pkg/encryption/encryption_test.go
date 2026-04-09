package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"testing"
)

func generateTestKeys(t *testing.T) (pubKeyB64 string, privKeyB64 string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("error generating RSA key: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("error marshaling public key: %v", err)
	}
	pubKeyB64 = base64.RawStdEncoding.EncodeToString(pubBytes)

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("error marshaling private key: %v", err)
	}
	privKeyB64 = base64.StdEncoding.EncodeToString(privBytes)

	return pubKeyB64, privKeyB64
}

func TestEncryptDecryptHybrid(t *testing.T) {
	pubKey, privKey := generateTestKeys(t)

	plaintext := "este es un mensaje de prueba para el esquema híbrido"
	encrypted, err := EncryptHybrid(plaintext, pubKey)
	if err != nil {
		t.Fatalf("EncryptHybrid error: %v", err)
	}
	if encrypted == "" {
		t.Fatal("EncryptHybrid returned empty string")
	}

	decrypted, err := DecryptHybrid(encrypted, privKey)
	if err != nil {
		t.Fatalf("DecryptHybrid error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptHybrid_EmptyString(t *testing.T) {
	result, err := EncryptHybrid("", "any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}

	result, err = DecryptHybrid("", "any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestEncryptDecryptHybrid_LargePayload(t *testing.T) {
	pubKey, privKey := generateTestKeys(t)

	largePayload := make([]byte, 10000)
	for i := range largePayload {
		largePayload[i] = byte(i % 256)
	}
	plaintext := base64.StdEncoding.EncodeToString(largePayload)

	encrypted, err := EncryptHybrid(plaintext, pubKey)
	if err != nil {
		t.Fatalf("EncryptHybrid error: %v", err)
	}

	decrypted, err := DecryptHybrid(encrypted, privKey)
	if err != nil {
		t.Fatalf("DecryptHybrid error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatal("decrypted large payload does not match original")
	}
}

func TestEncryptDecryptRSAOAEP(t *testing.T) {
	pubKey, privKey := generateTestKeys(t)

	plaintext := "token-corto"
	encrypted, err := EncryptRSAOAEP(plaintext, pubKey)
	if err != nil {
		t.Fatalf("EncryptRSAOAEP error: %v", err)
	}
	if encrypted == "" {
		t.Fatal("EncryptRSAOAEP returned empty string")
	}

	decrypted, err := DecryptRSAOAEP(encrypted, privKey)
	if err != nil {
		t.Fatalf("DecryptRSAOAEP error: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncryptDecryptRSAOAEP_EmptyString(t *testing.T) {
	result, err := EncryptRSAOAEP("", "any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}

	result, err = DecryptRSAOAEP("", "any-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestDecryptHybrid_InvalidFormat(t *testing.T) {
	_, err := DecryptHybrid("invalid-no-colon", "any-key")
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
}
