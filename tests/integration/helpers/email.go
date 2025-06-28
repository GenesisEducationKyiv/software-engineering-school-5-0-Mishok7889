package helpers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type MailHogMessage struct {
	ID      string  `json:"ID"`
	From    From    `json:"From"`
	To      []To    `json:"To"`
	Content Content `json:"Content"`
}

type From struct {
	Relays  interface{} `json:"Relays"`
	Mailbox string      `json:"Mailbox"`
	Domain  string      `json:"Domain"`
	Params  string      `json:"Params"`
}

type To struct {
	Relays  interface{} `json:"Relays"`
	Mailbox string      `json:"Mailbox"`
	Domain  string      `json:"Domain"`
	Params  string      `json:"Params"`
}

type Content struct {
	Headers map[string][]string `json:"Headers"`
	Body    string              `json:"Body"`
	Size    int                 `json:"Size"`
	MIME    string              `json:"MIME"`
}

type MailHogResponse struct {
	Total int              `json:"total"`
	Count int              `json:"count"`
	Start int              `json:"start"`
	Items []MailHogMessage `json:"items"`
}

func CheckEmailSent(to, subjectContains string) bool {
	resp, err := http.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		return false
	}
	defer func() {
		_ = resp.Body.Close() // Explicitly ignore close error
	}()

	var mailHogResp MailHogResponse
	if err := json.NewDecoder(resp.Body).Decode(&mailHogResp); err != nil {
		return false
	}

	for _, message := range mailHogResp.Items {
		for _, recipient := range message.To {
			recipientEmail := fmt.Sprintf("%s@%s", recipient.Mailbox, recipient.Domain)
			if recipientEmail != to {
				continue
			}

			subjects, exists := message.Content.Headers["Subject"]
			if !exists {
				continue
			}

			for _, subject := range subjects {
				if strings.Contains(subject, subjectContains) {
					return true
				}
			}
		}
	}

	return false
}

func ClearEmails() error {
	req, err := http.NewRequest("DELETE", "http://localhost:8025/api/v1/messages", nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close() // Explicitly ignore close error
	}()

	return nil
}

func GetEmailContent(to, subjectContains string) (string, error) {
	resp, err := http.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close() // Explicitly ignore close error
	}()

	var mailHogResp MailHogResponse
	if err := json.NewDecoder(resp.Body).Decode(&mailHogResp); err != nil {
		return "", err
	}

	for _, message := range mailHogResp.Items {
		for _, recipient := range message.To {
			recipientEmail := fmt.Sprintf("%s@%s", recipient.Mailbox, recipient.Domain)
			if recipientEmail != to {
				continue
			}

			subjects, exists := message.Content.Headers["Subject"]
			if !exists {
				continue
			}

			for _, subject := range subjects {
				if strings.Contains(subject, subjectContains) {
					return message.Content.Body, nil
				}
			}
		}
	}

	return "", fmt.Errorf("email not found")
}
