package email

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

const ErrNotConfiguredText = "email sending is not configured"

type Config struct {
	Host      string
	Port      string
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

type Message struct {
	To                 string
	Subject            string
	Body               string
	ReplyTo            string
	AttachmentFilename string
	AttachmentData     []byte
}

type Sender interface {
	Send(ctx context.Context, msg Message) error
}

type Service struct {
	cfg Config
}

func NewService(cfg Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Send(ctx context.Context, msg Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.validateMessage(msg); err != nil {
		return err
	}

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", net.JoinHostPort(s.cfg.Host, s.cfg.Port))
	if err != nil {
		return fmt.Errorf("connect smtp: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		return fmt.Errorf("create smtp client: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); !ok {
		return fmt.Errorf("smtp server does not support STARTTLS")
	}
	if err := client.StartTLS(&tls.Config{
		ServerName: s.cfg.Host,
		MinVersion: tls.VersionTLS12,
	}); err != nil {
		return fmt.Errorf("starttls: %w", err)
	}

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}

	if err := client.Mail(s.cfg.FromEmail); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}

	payload, err := buildMessage(s.cfg, msg)
	if err != nil {
		_ = w.Close()
		return err
	}
	if _, err := w.Write(payload); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}

func (s *Service) validateMessage(msg Message) error {
	if strings.TrimSpace(s.cfg.Host) == "" ||
		strings.TrimSpace(s.cfg.Port) == "" ||
		strings.TrimSpace(s.cfg.FromEmail) == "" {
		return fmt.Errorf(ErrNotConfiguredText)
	}
	if strings.TrimSpace(s.cfg.Username) != "" && strings.TrimSpace(s.cfg.Password) == "" {
		return fmt.Errorf(ErrNotConfiguredText)
	}
	if strings.TrimSpace(msg.To) == "" {
		return fmt.Errorf("recipient email is required")
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return fmt.Errorf("email subject is required")
	}
	if strings.TrimSpace(msg.Body) == "" {
		return fmt.Errorf("email body is required")
	}
	if strings.TrimSpace(msg.AttachmentFilename) == "" || len(msg.AttachmentData) == 0 {
		return fmt.Errorf("invoice PDF attachment is required")
	}
	return nil
}

func buildMessage(cfg Config, msg Message) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	boundary := writer.Boundary()

	fromDisplay := strings.TrimSpace(cfg.FromEmail)
	if name := strings.TrimSpace(cfg.FromName); name != "" {
		fromDisplay = mime.QEncoding.Encode("utf-8", name) + " <" + cfg.FromEmail + ">"
	}

	headers := []string{
		"MIME-Version: 1.0",
		"From: " + fromDisplay,
		"To: " + msg.To,
		"Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject),
		"Date: " + time.Now().Format(time.RFC1123Z),
		"Content-Type: multipart/mixed; boundary=" + quoteBoundary(boundary),
	}
	if replyTo := strings.TrimSpace(msg.ReplyTo); replyTo != "" {
		headers = append(headers, "Reply-To: "+replyTo)
	}
	headers = append(headers, "", "")
	if _, err := buf.WriteString(strings.Join(headers, "\r\n")); err != nil {
		return nil, fmt.Errorf("build email headers: %w", err)
	}

	bodyHeader := textproto.MIMEHeader{}
	bodyHeader.Set("Content-Type", "text/plain; charset=utf-8")
	bodyHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	bodyPart, err := writer.CreatePart(bodyHeader)
	if err != nil {
		return nil, fmt.Errorf("create email body part: %w", err)
	}
	qp := quotedprintable.NewWriter(bodyPart)
	if _, err := qp.Write([]byte(msg.Body)); err != nil {
		return nil, fmt.Errorf("write email body: %w", err)
	}
	if err := qp.Close(); err != nil {
		return nil, fmt.Errorf("close email body writer: %w", err)
	}

	escapedFilename := url.PathEscape(msg.AttachmentFilename)
	attachmentHeader := textproto.MIMEHeader{}
	attachmentHeader.Set("Content-Type", "application/pdf")
	attachmentHeader.Set("Content-Transfer-Encoding", "base64")
	attachmentHeader.Set(
		"Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, sanitizeHeaderFilename(msg.AttachmentFilename), escapedFilename),
	)
	attachmentHeader.Set(
		"Content-Type",
		fmt.Sprintf(`application/pdf; name="%s"; name*=UTF-8''%s`, sanitizeHeaderFilename(msg.AttachmentFilename), escapedFilename),
	)
	attachmentPart, err := writer.CreatePart(attachmentHeader)
	if err != nil {
		return nil, fmt.Errorf("create email attachment part: %w", err)
	}
	lineWriter := newBase64LineWriter(attachmentPart)
	b64 := base64.NewEncoder(base64.StdEncoding, lineWriter)
	if _, err := b64.Write(msg.AttachmentData); err != nil {
		return nil, fmt.Errorf("write email attachment: %w", err)
	}
	if err := b64.Close(); err != nil {
		return nil, fmt.Errorf("close email attachment encoder: %w", err)
	}
	if err := lineWriter.Close(); err != nil {
		return nil, fmt.Errorf("close email attachment line writer: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close email multipart writer: %w", err)
	}
	return buf.Bytes(), nil
}

func quoteBoundary(boundary string) string {
	return `"` + strings.ReplaceAll(boundary, `"`, "") + `"`
}

func sanitizeHeaderFilename(filename string) string {
	replacer := strings.NewReplacer(`\`, "_", `"`, "_", "\r", "", "\n", "")
	return replacer.Replace(filename)
}

type base64LineWriter struct {
	w      io.Writer
	line   [76]byte
	used   int
	closed bool
}

func newBase64LineWriter(w io.Writer) *base64LineWriter {
	return &base64LineWriter{w: w}
}

func (w *base64LineWriter) Write(p []byte) (int, error) {
	written := 0
	for _, b := range p {
		w.line[w.used] = b
		w.used++
		written++
		if w.used == len(w.line) {
			if err := w.flush(); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (w *base64LineWriter) flush() error {
	if w.used == 0 {
		return nil
	}
	if _, err := w.w.Write(w.line[:w.used]); err != nil {
		return err
	}
	if _, err := w.w.Write([]byte("\r\n")); err != nil {
		return err
	}
	w.used = 0
	return nil
}

func (w *base64LineWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	return w.flush()
}
