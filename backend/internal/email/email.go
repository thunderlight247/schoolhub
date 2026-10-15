// Package email provides the application's outbound email. Resend
// (https://resend.com) is the provider; RESEND_API_KEY enables it. Every
// send failure is logged, never fatal: a broken email provider must not take
// down account creation or announcements.
//
// Two classes of mail are sent today:
//   - Welcome: when an admin provisions a login account for a staff member
//     (M6 CreateAccount) so the recipient knows how to sign in.
//   - Announcements: when a school-wide announcement is published (M13),
//     directors/staff hear about it by email without opening the portal.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const resendAPIURL = "https://api.resend.com/emails"
const sendTimeout = 10 * time.Second

// Sender is the minimal contract the rest of the app depends on, so tests
// can stub it. A nil *Service is a valid no-op Sender.
type Sender interface {
	Send(ctx context.Context, msg Message) error
}

// Message is one outbound email.
type Message struct {
	To      string
	Subject string
	HTML    string
}

// Service posts to Resend's REST API. A nil Service (no API key) silently
// drops mail — local development and CI have no provider credentials.
type Service struct {
	apiKey     string
	from       string
	httpClient *http.Client
}

// New builds the email Service. An empty apiKey yields a disabled service
// (Send becomes a no-op) rather than an error: email is enhancement, never
// a hard dependency.
func New(apiKey, fromAddress string) *Service {
	return &Service{
		apiKey:     apiKey,
		from:       fromAddress,
		httpClient: &http.Client{Timeout: sendTimeout},
	}
}

// Enabled reports whether a provider is configured.
func (s *Service) Enabled() bool {
	return s != nil && s.apiKey != ""
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// Send delivers one message. Disabled service → nil (no email expected).
// Provider/network/log-parsing failures are logged and returned, but every
// call site treats mail as best-effort.
func (s *Service) Send(ctx context.Context, msg Message) error {
	if !s.Enabled() {
		return nil
	}

	payload, err := json.Marshal(resendPayload{
		From:    s.from,
		To:      []string{msg.To},
		Subject: msg.Subject,
		HTML:    msg.HTML,
	})
	if err != nil {
		return fmt.Errorf("email: encode payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendAPIURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("email: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("email: send to %s failed: %v", msg.To, err)
		return fmt.Errorf("email: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("email: resend rejected send to %s: status %d", msg.To, resp.StatusCode)
		return fmt.Errorf("email: provider returned status %d", resp.StatusCode)
	}
	return nil
}

// SendWelcome emails a newly provisioned account holder their school portal
// URL. The password is deliberately never included in email.
func (s *Service) SendWelcome(ctx context.Context, to, name, portalURL string) error {
	if !s.Enabled() {
		return nil
	}
	html := fmt.Sprintf(`
<div style="font-family:Arial,Helvetica,sans-serif;max-width:480px;margin:0 auto">
  <h2 style="color:#173832">Welcome to SchoolHub Academy, %s</h2>
  <p>Your school portal account has been created.</p>
  <p><a href="%s" style="background:#255a51;color:#fff;padding:10px 18px;border-radius:8px;text-decoration:none">Sign in to the portal</a></p>
  <p style="color:#64748b;font-size:12px">If you were not expecting this email, contact the school office.</p>
</div>`, name, portalURL)
	return s.Send(ctx, Message{To: to, Subject: "Your SchoolHub Academy account is ready", HTML: html})
}

// SendAnnouncement notifies one recipient of a newly published
// announcement.
func (s *Service) SendAnnouncement(ctx context.Context, to, title, body string) error {
	if !s.Enabled() {
		return nil
	}
	html := fmt.Sprintf(`
<div style="font-family:Arial,Helvetica,sans-serif;max-width:480px;margin:0 auto">
  <h2 style="color:#173832">SchoolHub Academy announcement</h2>
  <h3 style="color:#255a51">%s</h3>
  <p>%s</p>
  <p style="color:#64748b;font-size:12px">Sign in to the portal for the full notice board.</p>
</div>`, title, body)
	return s.Send(ctx, Message{To: to, Subject: "SchoolHub: " + title, HTML: html})
}
