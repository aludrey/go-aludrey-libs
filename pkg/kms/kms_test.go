package kms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// KmsKey de test
const kmsKey = "a4fa74ac-d313-47d0-a856-6723c013684f"

func TestKMS(t *testing.T) {
	texto := "HELLO WORLD"
	Ciphertext, _ := EncryptWithKMSkey(kmsKey, "us-east-2 ", []byte(texto))
	Decrypt, _ := DecryptWithKMSkey(kmsKey, "us-east-2 ", Ciphertext)

	assert.NotEmpty(t, Ciphertext)
	assert.Equal(t, texto, string(Decrypt))
}
