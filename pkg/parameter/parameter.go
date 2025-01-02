package parameter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aludrey/go-aludrey-libs/pkg/commons"
	"github.com/aludrey/go-aludrey-libs/pkg/entity"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

var (
	config *entity.Config = commons.GetConfig()
	sess   *session.Session
	client *ssm.SSM
)

func getSSMClient() (*ssm.SSM, error) {
	if sess == nil {
		paramSess, err := session.NewSession(&aws.Config{
			Region: aws.String(config.AwsRegion),
		})
		if err != nil {
			log.Printf("Error creating session: %v", err)
			return nil, err
		}
		sess = paramSess
	}
	if client == nil {
		client = ssm.New(sess)
	}
	return client, nil
}

/*
func UpdateParameter(evironment string, appName string, name string, value string) error {
	client, err := getSSMClient()
	if err != nil {
		log.Printf("Error getting ssm client: %v", err)
		return err
	}

	paramsPath := strings.ToLower(fmt.Sprintf("/%s/%s/%s", evironment, appName, name))

	params := &ssm.PutParameterInput{
		Name:      aws.String(paramsPath),
		Value:     aws.String(value),
		Type:      aws.String("SecureString"),
		Overwrite: aws.Bool(true),
	}

	_, err = client.PutParameter(params)
	if err != nil {
		fmt.Println("Error updating parameter:", err)
		return err
	}

	return nil
}

func DeleteParameter(evironment string, appName string, name string) error {
	client, err := getSSMClient()
	if err != nil {
		log.Printf("Error getting ssm client: %v", err)
		return err
	}

	paramsPath := strings.ToLower(fmt.Sprintf("/%s/%s/%s", evironment, appName, name))

	params := &ssm.DeleteParameterInput{
		Name: aws.String(paramsPath),
	}

	_, err = client.DeleteParameter(params)
	if err != nil {
		fmt.Println("Error deleting parameter:", err)
		return err
	}

	return nil
}
func CreateParameter(evironment string, appName string, name string, value string) error {
	client, err := getSSMClient()
	if err != nil {
		log.Printf("Error getting ssm client: %v", err)
		return err
	}

	paramsPath := strings.ToLower(fmt.Sprintf("/%s/%s/%s", evironment, appName, name))

	params := &ssm.PutParameterInput{
		Name:      aws.String(paramsPath),
		Value:     aws.String(value),
		Type:      aws.String("SecureString"),
		Overwrite: aws.Bool(true),
	}

	_, err = client.PutParameter(params)
	if err != nil {
		fmt.Println("Error creating parameter:", err)
		return err
	}

	return nil
}
*/

func LoadParameters(evironment string, appName string) error {
	client, err := getSSMClient()
	if err != nil {
		log.Printf("Error getting ssm client: %v", err)
		return err
	}

	paramsPath := strings.ToLower("/" + evironment + "/" + appName)

	params := &ssm.GetParametersByPathInput{
		Path:           aws.String(paramsPath),
		Recursive:      aws.Bool(true),
		WithDecryption: aws.Bool(true),
		MaxResults:     aws.Int64(10),
	}

	result, err := client.GetParametersByPath(params)
	if err != nil {
		fmt.Println("Error getting parameters by path:", err)
		return err
	}
	for result.Parameters != nil && len(result.Parameters) > 0 {
		for _, param := range result.Parameters {
			os.Setenv(strings.ToUpper(filepath.Base(*param.Name)), *param.Value)
		}
		if result.NextToken == nil {
			break
		}
		params.NextToken = result.NextToken
		result, err = client.GetParametersByPath(params)
		if err != nil {
			fmt.Println("Error getting parameters by path:", err)
			return err
		}
	}

	return nil
}
