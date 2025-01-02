package queue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSendGetDeleteMessageE2E(t *testing.T) {
	// Crear parámetros de prueba
	queueURL := "https://sqs.us-east-2.amazonaws.com/253056462088/aludrey-dev-sqs-test"
	message := "Hello, SQS!"
	maxNumberOfMessages := int64(1)
	visibilityTimeout := int64(3)
	delaySeconds := int64(0)

	// Ejecutar la función
	err := SendMessage(queueURL, message, delaySeconds)

	// Verificar que no haya error
	assert.Nil(t, err, "Se esperaba que no hubiera error")

	time.Sleep(5 * time.Second) // wait

	// Ejecutar la función
	messages, err := GetMessages(queueURL, maxNumberOfMessages, visibilityTimeout)

	// Verificar que no haya error
	assert.Nil(t, err, "Se esperaba que no hubiera error")

	// Verificar que el mensaje no sea nulo
	assert.NotNil(t, messages, "Se esperaba que el mensaje no fuera nulo")

	// Verificar que el mensaje no sea vacío
	assert.NotEmpty(t, messages, "Se esperaba que el mensaje no fuera vacío")

	// Verificar que el mensaje sea igual al mensaje enviado
	assert.Equal(t, message, *messages[0].Body, "Se esperaba que el mensaje fuera igual al mensaje enviado")

	// Guardo el ReceiptHandle
	receiptHandle := *messages[0].ReceiptHandle

	time.Sleep(4 * time.Second) // wait

	// Elimino el mensaje
	err = DeleteMessage(queueURL, receiptHandle)

	// Verificar que no haya error
	assert.Nil(t, err, "Se esperaba que no hubiera error")
}
