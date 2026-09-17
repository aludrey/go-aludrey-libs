package bigquery

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/aludrey/go-aludrey-libs/pkg/secret"
)

var (
	ErrSecretMissingFilename = errors.New("bigquery: credentials secret is missing filename")
	ErrSecretMissingContent  = errors.New("bigquery: credentials secret is missing content")
)

// secretReader es la firma de secret.GetSecretValue, inyectable en tests.
type secretReader func(environment string, appName string, secretName string, region string) (string, error)

// CredentialsFromSecret lee de AWS Secrets Manager (vía pkg/secret) el secreto
// con el key del service account y devuelve el JSON del key listo para
// NewProvider. El payload esperado es {"filename": ..., "content": ...};
// filename se valida pero no se expone, porque el key nunca toca disco.
//
// Es deliberadamente una función aparte del constructor: el consumidor decide
// la política de fallo (p. ej. loguear y seguir arrancando para responder el
// healthcheck) y el provider queda construible en tests sin AWS.
func CredentialsFromSecret(environment string, appName string, secretName string, region string) (string, error) {
	return credentialsFromSecret(environment, appName, secretName, region, secret.GetSecretValue)
}

func credentialsFromSecret(environment string, appName string, secretName string, region string, read secretReader) (string, error) {
	payload, err := read(environment, appName, secretName, region)
	if err != nil {
		return "", fmt.Errorf("bigquery: reading credentials secret: %w", err)
	}
	return parseCredentialsPayload(payload)
}

// parseCredentialsPayload extrae el content del payload {filename, content}.
func parseCredentialsPayload(payload string) (string, error) {
	var parsed struct {
		Filename string `json:"filename"`
		Content  string `json:"content"`
	}
	if err := json.Unmarshal([]byte(payload), &parsed); err != nil {
		return "", fmt.Errorf("bigquery: parsing credentials secret: %w", err)
	}
	if strings.TrimSpace(parsed.Filename) == "" {
		return "", ErrSecretMissingFilename
	}
	content := strings.TrimSpace(parsed.Content)
	if content == "" {
		return "", ErrSecretMissingContent
	}
	return content, nil
}
