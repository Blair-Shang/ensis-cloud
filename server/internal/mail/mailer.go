package mail

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/ensis-cloud/server/internal/config"
)

type Mailer struct {
	cfg config.Config
	log *zap.Logger
}

func New(cfg config.Config, log *zap.Logger) *Mailer {
	if log == nil {
		log = zap.NewNop()
	}
	return &Mailer{cfg: cfg, log: log}
}

func (m *Mailer) SendOTP(email, purpose, code string) error {
	copy := otpCopyFor(purpose)
	ttlMin := int(m.cfg.OTPTTL.Minutes())
	if ttlMin <= 0 {
		ttlMin = 5
	}
	plain := renderOTPPlain(purpose, code, ttlMin)
	htmlBody := renderOTPHTML(purpose, code, ttlMin)

	smtpHost := strings.TrimSpace(m.cfg.SMTPHost)
	channel := "smtp"
	if smtpHost == "" {
		channel = "log"
	}

	fields := []zap.Field{
		zap.String("email", email),
		zap.String("purpose", purpose),
		zap.String("channel", channel),
	}
	switch {
	case m.cfg.MailLogOTP && smtpHost == "":
		fields = append(fields, zap.String("code", code))
	case m.cfg.MailLogOTP:
		fields = append(fields, zap.String("code_suffix", otpSuffix(code)))
	}
	m.log.Info("otp issued", fields...)

	if smtpHost == "" {
		return nil
	}
	return m.sendHTML(email, copy.Subject, plain, htmlBody)
}

func otpSuffix(code string) string {
	code = strings.TrimSpace(code)
	if len(code) <= 2 {
		return code
	}
	return strings.Repeat("*", len(code)-2) + code[len(code)-2:]
}

// SendFeedbackReply 通知用户：运营已回复其反馈工单。
func (m *Mailer) SendFeedbackReply(email, title, reply string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}
	subject := "Ensis 反馈回复：" + title
	body := fmt.Sprintf("您好，\n\n您提交的反馈「%s」已有回复：\n\n%s\n\n— Ensis 团队\n", title, reply)

	if strings.TrimSpace(m.cfg.SMTPHost) == "" {
		m.log.Info("feedback reply mail skipped",
			zap.String("email", email),
			zap.String("title", title),
			zap.String("reason", "no_smtp"),
		)
		return nil
	}
	if err := m.sendPlain(email, subject, body); err != nil {
		return err
	}
	m.log.Info("feedback reply mail sent", zap.String("email", email), zap.String("title", title))
	return nil
}

// SendTest 发送测试邮件（运维联调用）。
func (m *Mailer) SendTest(to, subject, body string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("empty recipient")
	}
	if strings.TrimSpace(m.cfg.SMTPHost) == "" {
		return fmt.Errorf("smtp host not configured")
	}
	if subject == "" {
		subject = "Ensis 邮件测试"
	}
	if body == "" {
		body = "这是一封来自 ensis-cloud 的 SMTP 测试邮件。"
	}
	return m.sendPlain(to, subject, body)
}

// SendOTPPreview 发送验证码模板样例（不写入 OTP 库，仅测样式）。
func (m *Mailer) SendOTPPreview(to, purpose, code string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return fmt.Errorf("empty recipient")
	}
	if strings.TrimSpace(m.cfg.SMTPHost) == "" {
		return fmt.Errorf("smtp host not configured")
	}
	if strings.TrimSpace(code) == "" {
		code = "834291"
	}
	if strings.TrimSpace(purpose) == "" {
		purpose = "register"
	}
	copy := otpCopyFor(purpose)
	ttlMin := int(m.cfg.OTPTTL.Minutes())
	if ttlMin <= 0 {
		ttlMin = 5
	}
	return m.sendHTML(to, copy.Subject, renderOTPPlain(purpose, code, ttlMin), renderOTPHTML(purpose, code, ttlMin))
}

func (m *Mailer) sendPlain(to, subject, body string) error {
	return m.deliver(to, subject, buildPlainMessage(m.fromAddr(), to, subject, body))
}

func (m *Mailer) sendHTML(to, subject, plain, htmlBody string) error {
	return m.deliver(to, subject, buildMultipartMessage(m.fromAddr(), to, subject, plain, htmlBody))
}

func (m *Mailer) fromAddr() string {
	from := strings.TrimSpace(m.cfg.SMTPFrom)
	if from == "" {
		from = strings.TrimSpace(m.cfg.SMTPUser)
	}
	return from
}

func (m *Mailer) deliver(to, subject string, msg []byte) error {
	_ = subject
	host := strings.TrimSpace(m.cfg.SMTPHost)
	if host == "" {
		return fmt.Errorf("smtp host empty")
	}
	port := m.cfg.SMTPPort
	if port <= 0 {
		port = 465
	}
	from := m.fromAddr()
	user := strings.TrimSpace(m.cfg.SMTPUser)
	pass := m.cfg.SMTPPassword
	addr := fmt.Sprintf("%s:%d", host, port)

	var auth smtp.Auth
	if user != "" {
		auth = smtp.PlainAuth("", user, pass, host)
	}

	if port == 465 {
		return sendMailTLS(addr, host, auth, from, []string{to}, msg)
	}
	if err := smtp.SendMail(addr, auth, from, []string{to}, msg); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}

func buildPlainMessage(from, to, subject, body string) []byte {
	headers := []string{
		"From: " + formatFrom(from),
		"To: " + to,
		"Subject: " + encodeSubject(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: base64",
		"Date: " + time.Now().Format(time.RFC1123Z),
	}
	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + foldBase64(body))
}

func buildMultipartMessage(from, to, subject, plain, htmlBody string) []byte {
	boundary := "nm_" + randomBoundary()
	headers := []string{
		"From: " + formatFrom(from),
		"To: " + to,
		"Subject: " + encodeSubject(subject),
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=\"" + boundary + "\"",
		"Date: " + time.Now().Format(time.RFC1123Z),
	}

	var b strings.Builder
	b.WriteString(strings.Join(headers, "\r\n"))
	b.WriteString("\r\n\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(foldBase64(plain))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "\r\n")
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(foldBase64(htmlBody))
	b.WriteString("\r\n")

	b.WriteString("--" + boundary + "--\r\n")
	return []byte(b.String())
}

func formatFrom(from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return "Ensis <noreply@niuma007.com>"
	}
	if strings.Contains(from, "<") {
		return from
	}
	return "Ensis <" + from + ">"
}

func foldBase64(s string) string {
	payload := base64.StdEncoding.EncodeToString([]byte(s))
	var folded strings.Builder
	for i := 0; i < len(payload); i += 76 {
		end := i + 76
		if end > len(payload) {
			end = len(payload)
		}
		folded.WriteString(payload[i:end])
		folded.WriteString("\r\n")
	}
	return folded.String()
}

func encodeSubject(subject string) string {
	if isASCII(subject) {
		return subject
	}
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(subject)) + "?="
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

func randomBoundary() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func sendMailTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	tlsCfg := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
	}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 20 * time.Second}, "tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}
	// MAIL FROM 使用纯地址，不带显示名
	mailFrom := from
	if i := strings.Index(from, "<"); i >= 0 {
		if j := strings.Index(from[i:], ">"); j >= 0 {
			mailFrom = strings.TrimSpace(from[i+1 : i+j])
		}
	}
	if err := c.Mail(mailFrom); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp rcpt %s: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		_ = w.Close()
		return fmt.Errorf("smtp write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp data close: %w", err)
	}
	if err := c.Quit(); err != nil {
		return fmt.Errorf("smtp quit: %w", err)
	}
	return nil
}
