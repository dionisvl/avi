package email

import (
	"strings"
	"testing"
)

func TestPickLocale(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"":   LocaleEN,
		"en": LocaleEN,
		"ru": LocaleRU,
		"de": LocaleEN,
		"RU": LocaleEN,
	}

	for input, want := range tests {
		if got := PickLocale(input); got != want {
			t.Fatalf("PickLocale(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestBuildVerificationMessage(t *testing.T) {
	t.Parallel()

	msg, err := buildVerificationMessage("ru", "123456")
	if err != nil {
		t.Fatal(err)
	}

	assertRenderedMessage(t, msg, "Подтверждение email — avi", []string{
		`<html lang="ru">`,
		"avi",
		"Подтверждение email",
		"Для подтверждения email введите код:",
		"123456",
		"Код действителен 24 часа.",
	}, []string{
		"Подтверждение email",
		"Для подтверждения email введите код:",
		"123456",
	})
}

func TestBuildVerificationMessageEN(t *testing.T) {
	t.Parallel()

	msg, err := buildVerificationMessage("en", "123456")
	if err != nil {
		t.Fatal(err)
	}

	assertRenderedMessage(t, msg, "Verify your email — avi", []string{
		`<html lang="en">`,
		"Verify your email",
		"Enter this code to verify your email:",
		"123456",
		"The code is valid for 24 hours.",
	}, []string{
		"Verify your email",
		"Enter this code to verify your email:",
		"123456",
	})
}

func TestBuildPasswordResetMessage(t *testing.T) {
	t.Parallel()

	msg, err := buildPasswordResetMessage("ru", "654321")
	if err != nil {
		t.Fatal(err)
	}

	assertRenderedMessage(t, msg, "Сброс пароля — avi", []string{
		`<html lang="ru">`,
		"Сброс пароля",
		"654321",
		"Код действителен 15 минут.",
	}, []string{
		"Сброс пароля",
		"654321",
		"Код действителен 15 минут.",
	})
}

func TestBuildPasswordResetMessageEN(t *testing.T) {
	t.Parallel()

	msg, err := buildPasswordResetMessage("en", "654321")
	if err != nil {
		t.Fatal(err)
	}

	assertRenderedMessage(t, msg, "Password reset — avi", []string{
		`<html lang="en">`,
		"Password reset",
		"654321",
		"The code is valid for 15 minutes.",
	}, []string{
		"Password reset",
		"654321",
		"The code is valid for 15 minutes.",
	})
}

func TestBuildContactMessage(t *testing.T) {
	t.Parallel()

	msg, err := buildContactMessage("ru", "Анна", "anna@example.com", "Вопрос", "Здравствуйте")
	if err != nil {
		t.Fatal(err)
	}

	assertRenderedMessage(t, msg, "Сообщение обратной связи — avi: Вопрос", []string{
		`<html lang="ru">`,
		"Сообщение обратной связи",
		"Новое сообщение из формы обратной связи.",
		"Анна",
		"anna@example.com",
		"Вопрос",
		"Здравствуйте",
	}, []string{
		"Сообщение обратной связи",
		"Имя: Анна",
		"Email: anna@example.com",
		"Тема: Вопрос",
		"Сообщение:",
		"Здравствуйте",
	})
}

func TestBuildContactMessageEN(t *testing.T) {
	t.Parallel()

	msg, err := buildContactMessage("en", "Anna", "anna@example.com", "Question", "Hello")
	if err != nil {
		t.Fatal(err)
	}

	assertRenderedMessage(t, msg, "Contact form message — avi: Question", []string{
		`<html lang="en">`,
		"Contact form message",
		"New contact form message.",
		"Anna",
		"anna@example.com",
		"Question",
		"Hello",
	}, []string{
		"Contact form message",
		"Name: Anna",
		"Email: anna@example.com",
		"Subject: Question",
		"Message:",
		"Hello",
	})
}

func TestRenderedEmailEscapesHTMLContent(t *testing.T) {
	t.Parallel()

	body := `<script>alert("x")</script>`
	msg, err := buildContactMessage("en", `<b>Alice</b>`, "alice@example.com", "Hello\nWorld", body)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(msg.html, "&lt;b&gt;Alice&lt;/b&gt;") {
		t.Fatalf("sender name was not escaped:\n%s", msg.html)
	}
	if !strings.Contains(msg.html, "&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt;") {
		t.Fatalf("message body was not escaped:\n%s", msg.html)
	}
	if strings.Contains(msg.html, body) {
		t.Fatalf("raw script body found in HTML:\n%s", msg.html)
	}
	if msg.subject != "Contact form message — avi: Hello World" {
		t.Fatalf("unexpected sanitized subject: %q", msg.subject)
	}
}

func TestContactMessageSubjectOmitsEmptySuffix(t *testing.T) {
	t.Parallel()

	if got := buildContactMessageSubject("en", " \n "); got != "Contact form message — avi" {
		t.Fatalf("unexpected subject: %q", got)
	}
}

func assertRenderedMessage(t *testing.T, msg renderedMessage, wantSubject string, wantHTML []string, wantText []string) {
	t.Helper()

	if msg.subject != wantSubject {
		t.Fatalf("subject = %q, want %q", msg.subject, wantSubject)
	}
	if strings.TrimSpace(msg.html) == "" {
		t.Fatal("HTML body is empty")
	}
	if strings.TrimSpace(msg.text) == "" {
		t.Fatal("text body is empty")
	}
	for _, want := range wantHTML {
		if !strings.Contains(msg.html, want) {
			t.Fatalf("HTML does not contain %q:\n%s", want, msg.html)
		}
	}
	for _, want := range wantText {
		if !strings.Contains(msg.text, want) {
			t.Fatalf("text does not contain %q:\n%s", want, msg.text)
		}
	}
}
