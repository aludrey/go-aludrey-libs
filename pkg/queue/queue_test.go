package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSQSClient(t *testing.T) {
	// Ejecutar la función
	err := createSQSClient()

	// Verificar que no haya error
	assert.Nil(t, err, "Se esperaba que no hubiera error")
}

func TestDeleteMessageError(t *testing.T) {
	// Configurar el contexto de prueba

	// Crear un mensaje de prueba
	queueURL := "mockQueueURL"
	receiptHandle := "mockReceiptHandle"

	// Ejecutar la función
	err := DeleteMessage(queueURL, receiptHandle)

	// Verificar que no haya error
	assert.NotNil(t, err, "Se esperaba que hubiera error")
}

func TestSendMessageError(t *testing.T) {
	// Crear un mensaje de prueba
	queueURL := "mockQueueURL"
	message := "Hello, SQS!"
	delaySeconds := int64(0)

	// Ejecutar la función
	err := SendMessage(queueURL, message, delaySeconds)

	// Verificar que no haya error
	assert.NotNil(t, err, "Se esperaba que hubiera error")
}

func TestGetMessages(t *testing.T) {

	// Crear parámetros de prueba
	queueURL := "mockQueueURL"
	maxNumberOfMessages := int64(5)
	visibilityTimeout := int64(10)

	// Ejecutar la función
	messages, err := GetMessages(queueURL, maxNumberOfMessages, visibilityTimeout)

	// Verificar que haya error
	assert.NotNil(t, err, "Se esperaba que hubiera error")

	// Verificar que el slice de mensajes sea nulo
	assert.Nil(t, messages, "Se esperaba que el slice de mensajes no fuera nulo")
}

func TestGetQueueAttributes(t *testing.T) {
	// Crear parámetros de prueba
	queueURL := "aludrey-dev-gestion-us-e2-kyc"

	err := SendMessage(queueURL, "TEST mensaje, SQS!", 0)
	assert.Nil(t, err, "Se esperaba que no hubiera error al envíar mensaje")

	// Ejecutar la función
	cantMSg, err := GetMessagesAvailable(queueURL)

	assert.Greater(t, cantMSg, int64(0), "Se esperaba que la cantidad de mensajes fuera mayor a 0")

	// Verificar que haya error
	assert.Nil(t, err, "Se esperaba que no hubiera error")

}

func TestGetQueueAttributesError(t *testing.T) {
	// Crear parámetros de prueba
	queueURL := "mockQueueURL"

	// Ejecutar la función
	_, err := GetMessagesAvailable(queueURL)

	// Verificar que haya error
	assert.NotNil(t, err, "Se esperaba que hubiera error")
}
