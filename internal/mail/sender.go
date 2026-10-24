// Package mail 提供 SMTP 邮件发送与内置邮件模板。
package mail

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/pichub/pichub/internal/config"
	"github.com/pichub/pichub/internal/logger"
)

// 邮件模板类型。
const (
	KindVerify      = "verify"
	KindReset       = "reset"
	KindChangeEmail = "change_email"
)

// ErrDisabled 表示邮件功能未启用。
var ErrDisabled = errors.New("mail: disabled")

// Message 是一封待发送邮件。
type Message struct {
	To      string
	Subject string
	HTML    string
	Text    string
}

// Sender 是邮件发送抽象。
type Sender interface {
	Send(ctx context.Context, msg Message) error
	Enabled() bool
}

// TemplateData 是邮件模板数据。
type TemplateData struct {
	Title         string
	SiteName      string
	Username      string
	Email         string
	ActionURL     string
	ActionText    string
	Intro         string
	ExpireMinutes int
}

//go:embed templates/*.html
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

var subjects = map[string]string{
	KindVerify:      "验证您的邮箱",
	KindReset:       "重置您的密码",
	KindChangeEmail: "确认更换邮箱",
}

type disabledSender struct{}

func (disabledSender) Send(context.Context, Message) error { return ErrDisabled }

func (disabledSender) Enabled() bool { return false }

// New 根据配置创建邮件发送器，未启用时返回空实现。
func New(cfg config.MailConfig, log *logger.Logger) Sender {
	if !cfg.Enabled || cfg.Host == "" {
		return disabledSender{}
	}
	if cfg.FromEmail == "" {
		cfg.FromEmail = cfg.Username
	}
	if cfg.Port == 0 {
		cfg.Port = 465
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	return &smtpSender{cfg: cfg, log: log}
}

// Build 根据模板类型构造邮件内容。
func Build(kind, to string, data TemplateData) (Message, error) {
	if data.SiteName == "" {
		data.SiteName = "PicHub"
	}
	subject, ok := subjects[kind]
	if !ok {
		return Message{}, fmt.Errorf("mail: unknown template %q", kind)
	}
	if data.Intro == "" {
		switch kind {
		case KindVerify:
			data.Intro = "感谢注册，请点击下方按钮完成邮箱验证。"
		case KindReset:
			data.Intro = "我们收到了重置密码的请求，请点击下方按钮设置新密码。"
		case KindChangeEmail:
			data.Intro = "请确认将邮箱更换为 " + data.Email + "。"
		}
	}
	if data.Title == "" {
		data.Title = subject
	}
	if data.ActionText == "" {
		data.ActionText = "点击继续"
	}
	if data.ExpireMinutes <= 0 {
		data.ExpireMinutes = 30
	}
	if data.Username == "" {
		data.Username = strings.Split(to, "@")[0]
	}
	var sb strings.Builder
	if err := templates.ExecuteTemplate(&sb, "layout", data); err != nil {
		return Message{}, fmt.Errorf("render mail template: %w", err)
	}
	return Message{
		To:      to,
		Subject: fmt.Sprintf("【%s】%s", data.SiteName, subject),
		HTML:    sb.String(),
		Text:    fmt.Sprintf("%s\n\n%s\n\n%s\n\n链接 %d 分钟内有效。如非本人操作请忽略本邮件。", data.Title, data.Intro, data.ActionURL, data.ExpireMinutes),
	}, nil
}
