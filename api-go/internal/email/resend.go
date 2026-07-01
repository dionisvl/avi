package email

import (
	"context"
	"log/slog"
	"strings"

	resendgo "github.com/resend/resend-go/v3"
)

// Sender is the interface implemented by both SMTPSender and ResendSender.
type Sender interface {
	SendVerificationCode(ctx context.Context, locale, to, code string) error
	SendPasswordResetCode(ctx context.Context, locale, to, code string) error
	SendContactMessage(ctx context.Context, locale, to, senderName, senderEmail, subject, message string) error
}

// New returns a ResendSender if host contains "resend", otherwise an SMTPSender.
func New(host string, port int, from, user, pass, frontendDomain string, logger *slog.Logger) Sender {
	if containsResend(host) {
		return &ResendSender{
			apiKey:         pass,
			from:           from,
			frontendDomain: frontendDomain,
			logger:         logger,
		}
	}

	return NewSMTPSender(host, port, from, user, pass, frontendDomain, logger)
}

func containsResend(host string) bool {
	return strings.Contains(host, "resend")
}

// ResendSender sends emails via the Resend REST API.
type ResendSender struct {
	apiKey         string
	from           string
	frontendDomain string
	logger         *slog.Logger
}

func (r *ResendSender) SendVerificationCode(ctx context.Context, locale, to, code string) error {
	msg, err := buildVerificationMessage(locale, code)
	if err != nil {
		return err
	}

	return r.send(ctx, to, msg)
}

func (r *ResendSender) SendPasswordResetCode(ctx context.Context, locale, to, code string) error {
	msg, err := buildPasswordResetMessage(locale, code)
	if err != nil {
		return err
	}
	return r.send(ctx, to, msg)
}

func (r *ResendSender) SendContactMessage(ctx context.Context, locale, to, senderName, senderEmail, subject, message string) error {
	msg, err := buildContactMessage(locale, senderName, senderEmail, subject, message)
	if err != nil {
		return err
	}
	return r.send(ctx, to, msg)
}

func (r *ResendSender) send(ctx context.Context, to string, msg renderedMessage) error {
	client := resendgo.NewClient(r.apiKey)
	params := &resendgo.SendEmailRequest{
		From:    r.from,
		To:      []string{to},
		Subject: msg.subject,
		Html:    msg.html,
		Text:    msg.text,
	}
	resp, err := client.Emails.SendWithContext(ctx, params)
	attrs := []any{
		slog.String("to", to),
		slog.String("subject", msg.subject),
		slog.String("from", r.from),
		slog.String("transport", "resend"),
	}
	if err != nil {
		r.logger.Error("email send failed", append(attrs, slog.String("error", err.Error()))...)
	} else {
		r.logger.Info("email sent", append(attrs, slog.String("id", resp.Id))...)
	}
	return err
}
