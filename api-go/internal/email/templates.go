package email

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"time"
)

type renderedMessage struct {
	subject string
	html    string
	text    string
}

type templateData struct {
	Lang         string
	Title        string
	Paragraphs   []string
	Code         string
	DetailsTitle string
	Details      []detailRow
	Message      *messageBlock
	CTA          *ctaBlock
	Note         string
	Year         int
}

type detailRow struct {
	Label string
	Value string
}

type messageBlock struct {
	Title string
	Body  string
}

type ctaBlock struct {
	Label string
	URL   string
}

type localizedEmailCopy struct {
	subject string
	title   string
	intro   string
	note    string
}

type contactMessageCopy struct {
	subject      string
	title        string
	intro        string
	nameLabel    string
	emailLabel   string
	subjectLabel string
	messageTitle string
}

const emailLayoutTemplate = `
<!DOCTYPE html>
<html lang="{{.Lang}}">
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: sans-serif; color: #333; max-width: 520px; margin: 0 auto; padding: 24px; line-height: 1.5; }
        h1 { color: #2563eb; margin: 0 0 4px; font-size: 28px; }
        h2 { color: #111827; margin: 24px 0 8px; font-size: 18px; }
        p { margin: 12px 0; }
        a { color: #2563eb; }
        .code { font-size: 32px; font-weight: bold; letter-spacing: 6px; color: #2563eb; font-family: monospace; margin: 24px 0; }
        .details { border-collapse: collapse; width: 100%; margin: 20px 0; }
        .details th { color: #6b7280; font-size: 12px; font-weight: 600; padding: 8px 12px 8px 0; text-align: left; vertical-align: top; width: 34%; }
        .details td { color: #111827; font-size: 14px; padding: 8px 0; vertical-align: top; }
        .message { background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 8px; margin: 20px 0; padding: 16px; white-space: pre-wrap; }
        .button { background: #2563eb; border-radius: 8px; color: #ffffff !important; display: inline-block; font-weight: 600; margin: 20px 0 8px; padding: 11px 18px; text-decoration: none; }
        .muted { color: #999; font-size: 12px; }
        footer { margin-top: 32px; color: #999; font-size: 12px; border-top: 1px solid #e5e7eb; padding-top: 16px; }
    </style>
</head>
<body>
    <h1>avi</h1>
    <h2>{{.Title}}</h2>
    {{range .Paragraphs}}<p>{{.}}</p>{{end}}
    {{if .Code}}<div class="code">{{.Code}}</div>{{end}}
    {{if .Details}}
    {{if .DetailsTitle}}<h2>{{.DetailsTitle}}</h2>{{end}}
    <table class="details">
        {{range .Details}}
        <tr>
            <th>{{.Label}}</th>
            <td>{{.Value}}</td>
        </tr>
        {{end}}
    </table>
    {{end}}
    {{if .Message}}
    <h2>{{.Message.Title}}</h2>
    <div class="message">{{.Message.Body}}</div>
    {{end}}
    {{if .CTA}}
    <p><a class="button" href="{{.CTA.URL}}">{{.CTA.Label}}</a></p>
    <p class="muted">{{.CTA.URL}}</p>
    {{end}}
    {{if .Note}}<p class="muted">{{.Note}}</p>{{end}}
    <footer>&copy; {{.Year}} avi</footer>
</body>
</html>
`

var emailLayout = template.Must(template.New("email_layout").Parse(emailLayoutTemplate))

var verificationCopy = map[string]localizedEmailCopy{
	LocaleRU: {
		subject: "Подтверждение email — avi",
		title:   "Подтверждение email",
		intro:   "Для подтверждения email введите код:",
		note:    "Код действителен 24 часа. Если вы не регистрировались — проигнорируйте письмо.",
	},
	LocaleEN: {
		subject: "Verify your email — avi",
		title:   "Verify your email",
		intro:   "Enter this code to verify your email:",
		note:    "The code is valid for 24 hours. If you did not sign up, ignore this email.",
	},
}

var passwordResetCopy = map[string]localizedEmailCopy{
	LocaleRU: {
		subject: "Сброс пароля — avi",
		title:   "Сброс пароля",
		intro:   "Введите этот код, чтобы продолжить восстановление пароля:",
		note:    "Код действителен 15 минут. Если вы не запрашивали сброс пароля — проигнорируйте письмо.",
	},
	LocaleEN: {
		subject: "Password reset — avi",
		title:   "Password reset",
		intro:   "Enter this code to continue resetting your password:",
		note:    "The code is valid for 15 minutes. If you did not request a password reset, ignore this email.",
	},
}

var contactMessageCopies = map[string]contactMessageCopy{
	LocaleRU: {
		subject:      "Сообщение обратной связи — avi",
		title:        "Сообщение обратной связи",
		intro:        "Новое сообщение из формы обратной связи.",
		nameLabel:    "Имя",
		emailLabel:   "Email",
		subjectLabel: "Тема",
		messageTitle: "Сообщение",
	},
	LocaleEN: {
		subject:      "Contact form message — avi",
		title:        "Contact form message",
		intro:        "New contact form message.",
		nameLabel:    "Name",
		emailLabel:   "Email",
		subjectLabel: "Subject",
		messageTitle: "Message",
	},
}

func buildVerificationMessage(locale, code string) (renderedMessage, error) {
	loc := PickLocale(locale)
	content := verificationCopy[loc]
	return renderEmail(content.subject, templateData{
		Lang:       loc,
		Title:      content.title,
		Paragraphs: []string{content.intro},
		Code:       code,
		Note:       content.note,
	}, buildText(content.title, []string{content.intro}, code, nil, nil, nil, content.note))
}

func buildPasswordResetMessage(locale, code string) (renderedMessage, error) {
	loc := PickLocale(locale)
	content := passwordResetCopy[loc]
	return renderEmail(content.subject, templateData{
		Lang:       loc,
		Title:      content.title,
		Paragraphs: []string{content.intro},
		Code:       code,
		Note:       content.note,
	}, buildText(content.title, []string{content.intro}, code, nil, nil, nil, content.note))
}

func buildContactMessageSubject(locale, subject string) string {
	base := contactMessageCopies[PickLocale(locale)].subject
	subject = cleanSubjectValue(subject)
	if subject == "" {
		return base
	}
	return base + ": " + subject
}

func buildContactMessage(locale, senderName, senderEmail, subject, message string) (renderedMessage, error) {
	loc := PickLocale(locale)
	content := contactMessageCopies[loc]
	cleanedSubject := cleanSubjectValue(subject)
	details := nonEmptyDetails([]detailRow{
		{Label: content.nameLabel, Value: senderName},
		{Label: content.emailLabel, Value: senderEmail},
		{Label: content.subjectLabel, Value: cleanedSubject},
	})
	msgBlock := &messageBlock{Title: content.messageTitle, Body: strings.TrimSpace(message)}
	paragraphs := []string{content.intro}
	msgSubject := buildContactMessageSubject(locale, subject)

	return renderEmail(msgSubject, templateData{
		Lang:       loc,
		Title:      content.title,
		Paragraphs: paragraphs,
		Details:    details,
		Message:    msgBlock,
	}, buildText(content.title, paragraphs, "", details, msgBlock, nil, ""))
}

func renderEmail(subject string, data templateData, text string) (renderedMessage, error) {
	if data.Year == 0 {
		data.Year = time.Now().Year()
	}

	var htmlBuf bytes.Buffer
	if err := emailLayout.Execute(&htmlBuf, data); err != nil {
		return renderedMessage{}, fmt.Errorf("render email template: %w", err)
	}

	return renderedMessage{
		subject: subject,
		html:    htmlBuf.String(),
		text:    text,
	}, nil
}

func buildText(title string, paragraphs []string, code string, details []detailRow, msgBlock *messageBlock, cta *ctaBlock, note string) string {
	var b strings.Builder
	writeTextLine(&b, title)

	for _, paragraph := range paragraphs {
		writeTextLine(&b, paragraph)
	}

	if code != "" {
		writeTextLine(&b, code)
	}

	for _, row := range details {
		writeTextLine(&b, row.Label+": "+row.Value)
	}

	if msgBlock != nil {
		writeTextLine(&b, msgBlock.Title+":")
		writeTextLine(&b, msgBlock.Body)
	}

	if cta != nil {
		writeTextLine(&b, cta.Label+": "+cta.URL)
	}

	if note != "" {
		writeTextLine(&b, note)
	}

	return strings.TrimSpace(b.String())
}

func writeTextLine(b *strings.Builder, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString(value)
}

func nonEmptyDetails(rows []detailRow) []detailRow {
	out := make([]detailRow, 0, len(rows))
	for _, row := range rows {
		row.Value = strings.TrimSpace(row.Value)
		if row.Value == "" {
			continue
		}
		out = append(out, row)
	}
	return out
}

func cleanSubjectValue(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.Join(strings.Fields(value), " ")
}
