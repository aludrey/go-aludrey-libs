package kms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// kmsKeyAlias es la clave de test de la cuenta de las credenciales en uso; por alias para no fijar el ID de
// una clave de una cuenta puntual.
const kmsKeyAlias = "alias/aludrey-dev-libs-test"

func TestKMS(t *testing.T) {
	texto := "HELLO WORLD"
	ciphertext, err := EncryptWithKMSkey(kmsKeyAlias, "us-east-2", []byte(texto))
	assert.NoError(t, err)
	decrypted, err := DecryptWithKMSkey(kmsKeyAlias, "us-east-2", ciphertext)
	assert.NoError(t, err)

	assert.NotEmpty(t, ciphertext)
	assert.Equal(t, texto, string(decrypted))
}
