package commons

import (
	"errors"
	"log"
	"os"

	"github.com/aludrey/go-aludrey-libs/pkg/entity"
	"github.com/google/uuid"
)

func GenerateId() string {
	uuidString := uuid.New()
	uuid := uuidString.String()
	return uuid
}

var config *entity.Config

func GetConfig() *entity.Config {
	conf, err := getConfig()
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return conf
}

func LoadEnvironment() (string, error) {
	environment, present := os.LookupEnv("ENV")
	if present && environment != "" {
		return environment, nil
	}
	err := errors.New("variable de ambiente ENV no puede ser nula")
	return "", err

}

func getConfig() (*entity.Config, error) {

	if config != nil {
		return config, nil
	}

	return loadConfig()
}

func loadConfig() (*entity.Config, error) {
	config = &entity.Config{}

	// Set default Value
	config.AwsRegion = "us-east-2"
	config.BucketTest = "aludrey-dev-bucket-test"
	config.QueueTest = "aludrey-dev-sqs-test"

	// Get values from environment variables
	aws_region, aws_region_present := os.LookupEnv("AWS_REGION")
	bucket_test_name, bucket_test_name_present := os.LookupEnv("BUCKET_TEST_NAME")
	queue_test_name, queue_test_name_present := os.LookupEnv("QUEUE_TEST_NAME")

	if aws_region_present {
		config.AwsRegion = aws_region
	}
	if bucket_test_name_present {
		config.BucketTest = bucket_test_name
	}
	if queue_test_name_present {
		config.QueueTest = queue_test_name
	}

	return config, nil
}
