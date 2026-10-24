package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/quotedprintable"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

// smtpSender 通过 SMTP 协议发送邮件，支持 SSL / STARTTLS。
type smtpSender struct {
	cfg config.MailConfig
	log *logger.Logger
}

func (s *smtpSender) Enabled() bool { return true }

// Send 发送邮件，按配置重试。
func (s *smtpSender) Send(ctx context.Context, msg Message) error {
	attempts := s.cfg.RetryTimes + 1
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.sendOnce(ctx, msg); err != nil {
			lastErr = err
			if s.log != nil {
				s.log.Warn("send mail failed", logger.Err(err), logger.String("to", msg.To), logger.Int("attempt", i+1))
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(i+1) * time.Second):
			}
			continue
		}
		return nil
	}
	return lastErr
}

func (s *smtpSender) sendOnce(ctx context.Context, msg Message) error {
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	client, err := s.dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if s.cfg.Username != "" {
		auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp auth: %w", err)
		}
	}
	from := s.cfg.FromEmail
	if from == "" {
		from = s.cfg.Username
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	body, err := s.buildMIME(from, msg)
	if err != nil {
		w.Close()
		return err
	}
	if _, err := w.Write(body); err != nil {
		w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return client.Quit()
}

// dial 根据加密方式建立连接。
func (s *smtpSender) dial(addr string) (*smtp.Client, error) {
	dialer := &net.Dialer{Timeout: s.cfg.Timeout}
	tlsCfg := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}
	switch strings.ToLower(s.cfg.Encryption) {
	case "ssl", "tls":
		conn, err := tls.DialWithDialer(dialer, "tcp", addr, tlsCfg)
		if err != nil {
			return nil, fmt.Errorf("smtp dial: %w", err)
		}
		client, err := smtp.NewClient(conn, s.cfg.Host)
		if err != nil {
			conn.Close()
			return nil, err
		}
		return client, nil
	case "starttls":
		client, err := smtp.Dial(addr)
		if err != nil {
			return nil, fmt.Errorf("smtp dial: %w", err)
		}
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				client.Close()
				return nil, fmt.Errorf("smtp starttls: %w", err)
			}
		}
		return client, nil
	default:
		client, err := smtp.Dial(addr)
		if err != nil {
			return nil, fmt.Errorf("smtp dial: %w", err)
		}
		return client, nil
	}
}

// buildMIME 构造 MIME 邮件正文。
func (s *smtpSender) buildMIME(from string, msg Message) ([]byte, error) {
	var buf bytes.Buffer
	fromName := s.cfg.FromName
	if fromName == "" {
		fromName = "PicHub"
	}
	buf.WriteString("From: " + fmt.Sprintf("%s <%s>", mime.BEncoding.Encode("utf-8", fromName), from) + "\r\n")
	buf.WriteString("To: " + msg.To + "\r\n")
	buf.WriteString("Subject: " + mime.QEncoding.Encode("utf-8", msg.Subject) + "\r\n")
	buf.WriteString("Date: " + time.Now().Format(time.RFC1123Z) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	if msg.Text == "" {
		buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&buf, msg.HTML); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}

	boundary := newBoundary()
	buf.WriteString("Content-Type: multipart/alternative; boundary=" + boundary + "\r\n\r\n")
	writePart := func(contentType, content string) error {
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: " + contentType + "; charset=UTF-8\r\n")
		buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")
		if err := writeQuotedPrintable(&buf, content); err != nil {
			return err
		}
		buf.WriteString("\r\n")
		return nil
	}
	if err := writePart("text/plain", msg.Text); err != nil {
		return nil, err
	}
	if err := writePart("text/html", msg.HTML); err != nil {
		return nil, err
	}
	buf.WriteString("--" + boundary + "--\r\n")
	return buf.Bytes(), nil
}

func writeQuotedPrintable(buf *bytes.Buffer, content string) error {
	w := quotedprintable.NewWriter(buf)
	if _, err := w.Write([]byte(content)); err != nil {
		return err
	}
	return w.Close()
}

func newBoundary() string {
	return "pichub-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
