package contact

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/dionisvl/avi/api-go/internal/email"
	apierr "github.com/dionisvl/avi/api-go/internal/errors"
)

type Input struct {
	Locale  string
	Name    string
	Email   string
	Subject string
	Message string
}

type Service interface {
	Send(ctx context.Context, in Input) error
}

type service struct {
	emailSender email.Sender
	recipientTo string
	logger      *slog.Logger
}

func New(emailSender email.Sender, recipientTo string, logger *slog.Logger) Service {
	return &service{
		emailSender: emailSender,
		recipientTo: strings.TrimSpace(recipientTo),
		logger:      logger,
	}
}

func (s *service) Send(ctx context.Context, in Input) error {
	if s.emailSender == nil || s.recipientTo == "" {
		return apierr.New(apierr.ErrInternal, "Contact delivery is not configured")
	}

	emailCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.emailSender.SendContactMessage(
		emailCtx,
		in.Locale,
		s.recipientTo,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Email),
		strings.TrimSpace(in.Subject),
		strings.TrimSpace(in.Message),
	); err != nil {
		s.logger.Error("send contact message", slog.String("to", s.recipientTo), slog.String("error", err.Error()))
		return apierr.New(apierr.ErrInternal, "Failed to send contact message")
	}

	return nil
}
