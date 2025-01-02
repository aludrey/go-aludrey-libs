package notification

import (
	"encoding/json"
	"testing"

	"github.com/aludrey/go-aludrey-libs/pkg/secret"
)

func TestNewGoogleChatNotifier(t *testing.T) {
	notifier := NewGoogleChatNotifier("https://chat.googleapis.com/123")
	if notifier == nil || notifier.(*GoogleChatNotifier).webhookURL != "https://chat.googleapis.com/123" {
		t.Errorf("webhook url not set properly")
	}
}

func TestNotify(t *testing.T) {
	type secretContentType struct {
		Value string `json:"value"`
	}

	var secretContent secretContentType
	secret, err := secret.GetSecretValue("dev", "go-aludrey-libs", "google_chat_webhook", "us-east-2")
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal([]byte(secret), &secretContent)
	if err != nil {
		panic(err)
	}
	g := NewGoogleChatNotifier(secretContent.Value)
	g.WithPrefix("Notification from aludrey:")
	err = g.Notify("Hello, World!")
	if err != nil {
		t.Errorf("failed to send notification")
	}

	// Test with empty webhook url
	g = NewGoogleChatNotifier("")
	err = g.Notify("Hello, World!")
	if err == nil {
		t.Errorf("Expected error")
	}
}

func TestWithPrefix(t *testing.T) {
	g := NewGoogleChatNotifier("https://chat.googleapis.com/123")
	g.WithPrefix("prefix")
	if g.(*GoogleChatNotifier).prefix != "prefix" {
		t.Errorf("prefix not set properly")
	}
}
