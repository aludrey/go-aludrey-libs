package secret

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go/aws"
)

func GetSecretValue(environment string, appName string, secretName string, region string) (string, error) {
	secretPath := strings.ToLower(fmt.Sprintf("/%s/%s/%s", environment, appName, secretName))

	config, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return "", err
	}

	log.Print("Getting secret from path: ", secretPath)

	// Create Secrets Manager client
	svc := secretsmanager.NewFromConfig(config)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretPath),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		log.Print(err.Error())
		return "", err
	}

	// Decrypts secret using the associated KMS key.
	var secretString string = *result.SecretString
	return secretString, nil
}
