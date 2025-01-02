package queue

import (
	"log"
	"strconv"

	"github.com/aludrey/go-aludrey-libs/pkg/commons"
	"github.com/aludrey/go-aludrey-libs/pkg/entity"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

var (
	sqsClient *sqs.SQS
	config    *entity.Config = commons.GetConfig()
)

func createSQSClient() error {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(config.AwsRegion)},
	)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		return err
	}

	sqsClient = sqs.New(sess)

	return nil
}

func validateInit() error {
	if sqsClient == nil {
		return createSQSClient()
	}

	return nil
}

func GetMessagesAvailable(queueURL string) (int64, error) {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return 0, err
	}

	// Get the number of messages available in the queue
	result, err := sqsClient.GetQueueAttributes(&sqs.GetQueueAttributesInput{
		QueueUrl:       aws.String(queueURL),
		AttributeNames: aws.StringSlice([]string{"ApproximateNumberOfMessages"}),
	})
	if err != nil {
		return 0, err
	}

	mensajes := *result.Attributes["ApproximateNumberOfMessages"]
	cantMsg, err := strconv.ParseInt(mensajes, 10, 64)
	if err != nil {
		return 0, err
	}

	return cantMsg, nil
}

func SendMessage(queueURL string, message string, delaySeconds int64) error {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return err
	}

	// Send message to SQS queue
	_, err = sqsClient.SendMessage(&sqs.SendMessageInput{
		MessageBody:  aws.String(message),
		QueueUrl:     &queueURL,
		DelaySeconds: aws.Int64(delaySeconds),
	})
	if err != nil {
		log.Println("Error sending message to SQS ", err)
		return err
	}

	return nil
}

func DeleteMessage(queueURL string, receiptHandle string) error {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return err
	}

	// Delete the message from the SQS queue
	input := &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	}

	_, err = sqsClient.DeleteMessage(input)
	if err != nil {
		return err
	}

	return nil
}

func GetMessages(queueURL string, maxNumberOfMessages int64, visibilityTimeout int64) ([]*sqs.Message, error) {
	err := validateInit()
	if err != nil {
		log.Printf("Error validating init: %v", err)
		return nil, err
	}

	// Receive messages from SQS queue
	result, err := sqsClient.ReceiveMessage(&sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(queueURL),
		MaxNumberOfMessages: aws.Int64(maxNumberOfMessages),
		VisibilityTimeout:   aws.Int64(visibilityTimeout),
	})
	if err != nil {
		log.Println("Error receiving message from SQS ", err)
		return nil, err
	}

	return result.Messages, err
}
