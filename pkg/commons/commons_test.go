package commons

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateId(t *testing.T) {
	id := GenerateId()
	assert.NotEmpty(t, id)
}

func TestGetConfig(t *testing.T) {
	config, err := getConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config)
}

func TestLoadConfig(t *testing.T) {
	os.Setenv("AWS_REGION", "us-east-2")
	os.Setenv("BUCKET_TEST_NAME", "aludrey-dev-bucket-test")
	os.Setenv("QUEUE_TEST_NAME", "aludrey-dev-sqs-test")
	os.Setenv("LOG_LEVEL", "4")
	os.Setenv("LOG_STDOUT", "true")
	os.Setenv("LOG_KINESIS", "1")
	os.Setenv("LOG_KINESIS_STREAM_NAME", "aludrey-dev-api-us-e2-front-scoring-mx")
	os.Setenv("APP_KEY", "app")
	os.Setenv("REQ_ID_KEY", "req_id")

	config, err := loadConfig()
	assert.Nil(t, err)
	assert.NotNil(t, config)
}

func TestLoadEnvironment(t *testing.T) {
	os.Setenv("ENV", "DEV")
	environment, err := LoadEnvironment()
	assert.Nil(t, err)
	assert.Equal(t, "DEV", environment)
}

func TestLoadEmptyEnvironment(t *testing.T) {
	os.Setenv("ENV", "")
	environment, err := LoadEnvironment()
	assert.NotNil(t, err)
	assert.Empty(t, environment)
}

func TestLoadNotPresentEnvironment(t *testing.T) {
	environment, err := LoadEnvironment()
	assert.NotNil(t, err)
	assert.Empty(t, environment)
}
