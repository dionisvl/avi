package email

import (
	"context"
	"log/slog"

	"github.com/wneessen/go-mail"
)

type SMTPSender struct {
	host           string
	port           int
	from           string
	user           string
	pass           string
	frontendDomain string
	logger         *slog.Logger
}

func NewSMTPSender(host string, port int, from, user, pass, frontendDomain string, logger *slog.Logger) *SMTPSender {
	return &SMTPSender{
		host:           host,
		port:           port,
		from:           from,
		user:           user,
		pass:           pass,
		frontendDomain: frontendDomain,
		logger:         logger,
	}
}

func (s *SMTPSender) SendVerificationCode(_ context.Context, locale, to, code string) error {
	msg, err := buildVerificationMessage(locale, code)
	if err != nil {
		return err
	}
	return s.send(to, msg)
}

func (s *SMTPSender) SendPasswordResetCode(_ context.Context, locale, to, code string) error {
	msg, err := buildPasswordResetMessage(locale, code)
	if err != nil {
		return err
	}
	return s.send(to, msg)
}

func (s *SMTPSender) SendContactMessage(_ context.Context, locale, to, senderName, senderEmail, subject, message string) error {
	msg, err := buildContactMessage(locale, senderName, senderEmail, subject, message)
	if err != nil {
		return err
	}
	return s.send(to, msg)
}

func (s *SMTPSender) newClient() (*mail.Client, error) {
	opts := []mail.Option{
		mail.WithPort(s.port),
	}
	if s.user != "" {
		if s.port == 465 {
			// Implicit SSL (port 465)
			opts = append(opts,
				mail.WithSSLPort(false),
				mail.WithSMTPAuth(mail.SMTPAuthPlain),
				mail.WithUsername(s.user),
				mail.WithPassword(s.pass),
			)
		} else {
			// STARTTLS (port 587 and others)
			opts = append(opts,
				mail.WithTLSPortPolicy(mail.TLSMandatory),
				mail.WithSMTPAuth(mail.SMTPAuthPlain),
				mail.WithUsername(s.user),
				mail.WithPassword(s.pass),
			)
		}
	} else {
		// No credentials (mailpit)
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}
	return mail.NewClient(s.host, opts...)
}

func (s *SMTPSender) send(to string, msg renderedMessage) error {
	m := mail.NewMsg()
	if err := m.From(s.from); err != nil {
		return err
	}
	if err := m.To(to); err != nil {
		return err
	}
	m.Subject(msg.subject)
	m.SetBodyString(mail.TypeTextPlain, msg.text)
	m.AddAlternativeString(mail.TypeTextHTML, msg.html)

	c, err := s.newClient()
	if err != nil {
		return err
	}
	err = c.DialAndSend(m)
	s.logResult(to, msg.subject, "smtp", err)
	return err
}

func (s *SMTPSender) logResult(to, subject, transport string, err error) {
	attrs := []any{
		slog.String("to", to),
		slog.String("subject", subject),
		slog.String("from", s.from),
		slog.String("transport", transport),
		slog.String("host", s.host),
	}
	if err != nil {
		s.logger.Error("email send failed", append(attrs, slog.String("error", err.Error()))...)
	} else {
		s.logger.Info("email sent", attrs...)
	}
}
