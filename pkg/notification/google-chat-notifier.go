package notification

import (
	"errors"
	"net/http"
	"strings"
)

type GoogleChatNotifier struct {
	webhookURL string
	prefix     string
}

func NewGoogleChatNotifier(webhookURL string) Notifier {
	return &GoogleChatNotifier{
		webhookURL: webhookURL,
	}
}

func (g *GoogleChatNotifier) Notify(msg string) error {
	response, err := http.Post(g.webhookURL, "application/json", strings.NewReader(`{"text": "`+g.prefix+" "+msg+`"}`))
	if err != nil {
		return err
	}
	if response.Status != "200 OK" {
		return errors.New("failed to send notification")
	}
	return nil
}

func (g *GoogleChatNotifier) WithPrefix(prefix string) {
	g.prefix = prefix
}
