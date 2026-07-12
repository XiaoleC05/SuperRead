package mailer

import (
	"fmt"
	"net/smtp"

	"github.com/XiaoleC05/SuperRead/internal/config"
)

type BriefingArticle struct {
	Title     string
	FeedTitle string
	Summary   string
	URL       string
}

func SendBriefing(to, subject, htmlBody string) error {
	cfg := config.Cfg
	if cfg.SMTPHost == "" {
		return fmt.Errorf("SMTP not configured")
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)

	msg := []byte("From: " + cfg.FromEmail + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		htmlBody)

	return smtp.SendMail(
		fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort),
		auth,
		cfg.FromEmail,
		[]string{to},
		msg,
	)
}