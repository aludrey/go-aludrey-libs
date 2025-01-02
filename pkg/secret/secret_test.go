package secret

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSecretError(t *testing.T) {
	secret, err := GetSecretValue("dev", "aludrey-app-test", "test-secret", "us-east-1")

	assert.True(t, secret == "")
	assert.NotNil(t, err)
}

func TestGetSecretValue(t *testing.T) {
	secret, err := GetSecretValue("dev", "aludrey-app-test", "test-secret", "us-east-2")
	assert.Nil(t, err)
	assert.True(t, secret != "")
}

func TestGetSecret(t *testing.T) {
	secret, err := GetSecretValue("dev", "aludrey-kyc-evidencia-big-query", "bigquery-key", "us-east-2")
	t.Log(secret)
	t.Log(err)
	assert.True(t, secret != "")
}
