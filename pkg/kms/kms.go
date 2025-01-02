package kms

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

func createKMSClient(region string) *kms.KMS {
	// Create AWS Session
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	// Return KMS client
	return kms.New(sess)
}

func DecryptWithKMSkey(kms_id string, aws_region string, encrypted_data []byte) ([]byte, error) {

	// Create KMS service client
	kms_service := createKMSClient(aws_region)

	// Decrypt the data
	result, err := kms_service.Decrypt(&kms.DecryptInput{
		CiphertextBlob: encrypted_data,
		KeyId:          aws.String(kms_id),
	})

	if err != nil {
		err = errors.New("Got error decrypting data: " + err.Error())
		return nil, err
	}

	return result.Plaintext, err
}

func EncryptWithKMSkey(kms_id string, aws_region string, decrypted_data []byte) ([]byte, error) {

	// Create KMS service client
	kms_service := createKMSClient(aws_region)

	// Encrypt the data
	result, err := kms_service.Encrypt(&kms.EncryptInput{
		KeyId:     aws.String(kms_id),
		Plaintext: decrypted_data,
	})

	if err != nil {
		err = errors.New("Got error encrypting data: " + err.Error())
		return nil, err
	}

	return result.CiphertextBlob, nil
}
