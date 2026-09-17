package bigquery

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const serviceAccountKey = `{"type":"service_account","project_id":"proj","private_key":"-----BEGIN PRIVATE KEY-----\nabc\n-----END PRIVATE KEY-----\n","client_email":"sa@proj.iam.gserviceaccount.com"}`

func TestCredentialsFromSecret(t *testing.T) {
	testCases := []struct {
		name        string
		payload     string
		readErr     error
		expected    string
		expectedErr error
		errContains string
	}{
		{
			name:     "returns only the key content",
			payload:  `{"filename":"key.json","content":` + jsonString(serviceAccountKey) + `}`,
			expected: serviceAccountKey,
		},
		{
			name:     "content is trimmed",
			payload:  `{"filename":"key.json","content":"  {\"type\":\"service_account\"}\n"}`,
			expected: `{"type":"service_account"}`,
		},
		{
			name:        "missing filename",
			payload:     `{"content":"{}"}`,
			expectedErr: ErrSecretMissingFilename,
		},
		{
			name:        "blank filename",
			payload:     `{"filename":"   ","content":"{}"}`,
			expectedErr: ErrSecretMissingFilename,
		},
		{
			name:        "missing content",
			payload:     `{"filename":"key.json"}`,
			expectedErr: ErrSecretMissingContent,
		},
		{
			name:        "blank content",
			payload:     `{"filename":"key.json","content":"  "}`,
			expectedErr: ErrSecretMissingContent,
		},
		{
			name:        "malformed payload",
			payload:     `not-json`,
			errContains: "parsing credentials secret",
		},
		{
			name:        "secrets manager error is wrapped",
			readErr:     errBoom,
			expectedErr: errBoom,
			errContains: "reading credentials secret",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var gotArgs []string
			read := func(environment, appName, secretName, region string) (string, error) {
				gotArgs = []string{environment, appName, secretName, region}
				return tc.payload, tc.readErr
			}

			got, err := credentialsFromSecret("dev", "gestion", "bigquery-key", "us-east-2", read)

			assert.Equal(t, []string{"dev", "gestion", "bigquery-key", "us-east-2"}, gotArgs)
			if tc.expectedErr != nil {
				assert.ErrorIs(t, err, tc.expectedErr)
			}
			if tc.errContains != "" {
				assert.ErrorContains(t, err, tc.errContains)
			}
			if tc.expectedErr == nil && tc.errContains == "" {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, got)
		})
	}
}

// jsonString devuelve s como literal JSON entre comillas.
func jsonString(s string) string {
	out := []byte{'"'}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			out = append(out, '\\', '"')
		case '\\':
			out = append(out, '\\', '\\')
		case '\n':
			out = append(out, '\\', 'n')
		default:
			out = append(out, s[i])
		}
	}
	return string(append(out, '"'))
}
