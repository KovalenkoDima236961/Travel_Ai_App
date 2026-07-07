package email

import (
	"context"
	"net/smtp"
	"strings"
	"testing"
)

func TestMaskEmail(t *testing.T) {
	cases := map[string]string{
		"anna@example.com": "an***@example.com",
		"a@example.com":    "***@example.com",
		"ab@example.com":   "***@example.com",
		"abc@example.com":  "ab***@example.com",
		"not-an-email":     "***",
		"":                 "***",
		"trailing@":        "***",
		"@leading.com":     "***",
	}
	for in, want := range cases {
		if got := MaskEmail(in); got != want {
			t.Errorf("MaskEmail(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmailMessageValidate(t *testing.T) {
	valid := EmailMessage{ToEmail: "a@b.com", Subject: "Hi", TextBody: "Body"}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected valid message, got %v", err)
	}

	cases := map[string]EmailMessage{
		"missing recipient": {Subject: "s", TextBody: "b"},
		"missing subject":   {ToEmail: "a@b.com", TextBody: "b"},
		"missing text":      {ToEmail: "a@b.com", Subject: "s"},
		"blank recipient":   {ToEmail: "   ", Subject: "s", TextBody: "b"},
	}
	for name, msg := range cases {
		t.Run(name, func(t *testing.T) {
			if err := msg.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", name)
			}
		})
	}
}

func TestMockSenderSendsValidMessage(t *testing.T) {
	sender := NewMockSender(nil)
	err := sender.Send(context.Background(), EmailMessage{
		ToEmail:  "anna@example.com",
		ToName:   "Anna",
		Subject:  "New comment on a trip",
		TextBody: "Open the trip: http://localhost:3000/trips/abc",
	})
	if err != nil {
		t.Fatalf("mock send returned error: %v", err)
	}
}

func TestMockSenderRejectsInvalidMessage(t *testing.T) {
	sender := NewMockSender(nil)
	if err := sender.Send(context.Background(), EmailMessage{Subject: "s", TextBody: "b"}); err == nil {
		t.Fatal("expected mock send to reject a message with no recipient")
	}
}

func TestNewSenderProviderSelection(t *testing.T) {
	mock, err := NewSender(Config{Provider: "mock"}, nil)
	if err != nil {
		t.Fatalf("mock provider: %v", err)
	}
	if _, ok := mock.(*MockSender); !ok {
		t.Fatalf("expected *MockSender, got %T", mock)
	}

	smtpSender, err := NewSender(Config{
		Provider: "smtp",
		SMTP:     SMTPConfig{Host: "smtp.example.com", FromEmail: "no-reply@example.com"},
	}, nil)
	if err != nil {
		t.Fatalf("smtp provider: %v", err)
	}
	if _, ok := smtpSender.(*SMTPSender); !ok {
		t.Fatalf("expected *SMTPSender, got %T", smtpSender)
	}

	if _, err := NewSender(Config{Provider: "carrier-pigeon"}, nil); err == nil {
		t.Fatal("expected unsupported provider to error")
	}
}

func TestNewSMTPSenderRequiresHost(t *testing.T) {
	if _, err := NewSMTPSender(SMTPConfig{FromEmail: "no-reply@example.com"}, nil); err == nil {
		t.Fatal("expected error when SMTP host is missing")
	}
	if _, err := NewSMTPSender(SMTPConfig{Host: "smtp.example.com"}, nil); err == nil {
		t.Fatal("expected error when SMTP from-email is missing")
	}
}

func TestBuildMIMEMessageTextOnly(t *testing.T) {
	raw, err := buildMIMEMessage(
		SMTPConfig{FromEmail: "no-reply@example.com", FromName: "AI Travel Planner"},
		EmailMessage{ToEmail: "anna@example.com", ToName: "Anna", Subject: "Trip updated", TextBody: "line one\nline two"},
	)
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}
	out := string(raw)
	for _, want := range []string{
		"From: \"AI Travel Planner\" <no-reply@example.com>",
		"To: \"Anna\" <anna@example.com>",
		"Subject: Trip updated",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=\"utf-8\"",
		"line one\r\nline two",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("MIME message missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "multipart") {
		t.Error("text-only message should not be multipart")
	}
}

func TestBuildMIMEMessageMultipart(t *testing.T) {
	raw, err := buildMIMEMessage(
		SMTPConfig{FromEmail: "no-reply@example.com", FromName: "AI Travel Planner"},
		EmailMessage{ToEmail: "anna@example.com", Subject: "Trip updated", TextBody: "plain text", HTMLBody: "<p>html</p>"},
	)
	if err != nil {
		t.Fatalf("buildMIMEMessage: %v", err)
	}
	out := string(raw)
	for _, want := range []string{
		"Content-Type: multipart/alternative; boundary=",
		"Content-Type: text/plain; charset=\"utf-8\"",
		"Content-Type: text/html; charset=\"utf-8\"",
		"plain text",
		"<p>html</p>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("multipart MIME missing %q\n---\n%s", want, out)
		}
	}
}

// TestSMTPSenderUsesTransport verifies the sender routes a valid message through
// its transport with the configured from/to and a non-empty body, without any
// real network call.
func TestSMTPSenderUsesTransport(t *testing.T) {
	sender, err := NewSMTPSender(SMTPConfig{Host: "smtp.example.com", Port: 587, FromEmail: "no-reply@example.com"}, nil)
	if err != nil {
		t.Fatalf("new smtp sender: %v", err)
	}

	var gotAddr, gotFrom string
	var gotTo []string
	var gotBody []byte
	sender.send = func(addr string, _ smtp.Auth, from string, to []string, msg []byte) error {
		gotAddr, gotFrom, gotTo, gotBody = addr, from, to, msg
		return nil
	}

	err = sender.Send(context.Background(), EmailMessage{
		ToEmail:  "anna@example.com",
		Subject:  "Trip updated",
		TextBody: "Open the trip",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if gotAddr != "smtp.example.com:587" {
		t.Errorf("unexpected addr %q", gotAddr)
	}
	if gotFrom != "no-reply@example.com" {
		t.Errorf("unexpected from %q", gotFrom)
	}
	if len(gotTo) != 1 || gotTo[0] != "anna@example.com" {
		t.Errorf("unexpected to %v", gotTo)
	}
	if !strings.Contains(string(gotBody), "Subject: Trip updated") {
		t.Errorf("body missing subject header: %s", gotBody)
	}
}
