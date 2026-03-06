package service

import (
	"fmt"
	"os"
	"time"

	"github.com/resend/resend-go/v2"
)

type EmailService struct {
	client resend.Client
	from   string
}

func NewEmailService() *EmailService {

	apiKey := os.Getenv("RESEND_API_KEY")
	from := os.Getenv("FROM_EMAIL")

	client := resend.NewClient(apiKey)

	return &EmailService{
		client: *client,
		from:   from,
	}
}

type EmailRequest struct {
	FromEmail string `json:"from_email"`
	To        string `json:"to"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
}

func (es *EmailService) SendMail(req EmailRequest) error {

	params := &resend.SendEmailRequest{
		From:    es.from,
		To:      []string{req.To},
		ReplyTo: req.FromEmail,
		Subject: req.Subject,
		Html:    req.Message,
	}

	_, err := es.client.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}
	return nil
}

func (e *EmailService) SendBulkMails(requests []EmailRequest) (successes, failures int) {

	rateLimitDelay := time.Millisecond * 500

	for i, req := range requests {
		if err := e.SendMail(req); err != nil {
			fmt.Printf("Failed to send to %s: %v\n", req.To, err)
			failures++
		} else {
			fmt.Printf("Email sent to %s\n", req.To)
			successes++
		}
		if i < len(requests)-1 {
			time.Sleep(rateLimitDelay)
		}
	}
	return
}
