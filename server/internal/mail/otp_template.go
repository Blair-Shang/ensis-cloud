package mail

import (
	"fmt"
	"html"
	"strings"
)

type otpCopy struct {
	Subject string
	Title   string
	Lead    string
	Action  string
}

func otpCopyFor(purpose string) otpCopy {
	switch strings.TrimSpace(purpose) {
	case "register":
		return otpCopy{
			Subject: "Ensis 注册验证码",
			Title:   "完成账户注册",
			Lead:    "你正在注册 Ensis 云账户。请使用以下验证码完成注册：",
			Action:  "注册验证",
		}
	case "login":
		return otpCopy{
			Subject: "Ensis 登录验证码",
			Title:   "验证码登录",
			Lead:    "你正在使用邮箱验证码登录或注册 Ensis。若该邮箱尚未注册，验证通过后将自动创建账户。请使用以下验证码：",
			Action:  "登录验证",
		}
	case "reset":
		return otpCopy{
			Subject: "Ensis 重置密码验证码",
			Title:   "重置账户密码",
			Lead:    "你正在重置 Ensis 云账户密码。请使用以下验证码继续：",
			Action:  "重置验证",
		}
	default:
		return otpCopy{
			Subject: "Ensis 验证码",
			Title:   "安全验证",
			Lead:    "你正在进行 Ensis 账户安全验证。请使用以下验证码：",
			Action:  "安全验证",
		}
	}
}

func renderOTPPlain(purpose, code string, ttlMin int) string {
	c := otpCopyFor(purpose)
	if ttlMin <= 0 {
		ttlMin = 5
	}
	return fmt.Sprintf(
		"%s\n\n%s\n\n验证码：%s\n有效期：%d 分钟\n\n如非本人操作，请忽略本邮件。\n\n— Ensis 团队\nhttps://www.niuma007.com\n",
		c.Title,
		c.Lead,
		code,
		ttlMin,
	)
}

func renderOTPHTML(purpose, code string, ttlMin int) string {
	c := otpCopyFor(purpose)
	if ttlMin <= 0 {
		ttlMin = 5
	}
	safeCode := html.EscapeString(strings.TrimSpace(code))
	safeTitle := html.EscapeString(c.Title)
	safeLead := html.EscapeString(c.Lead)
	safeAction := html.EscapeString(c.Action)

	// 邮件客户端兼容：table 布局 + 行内样式，避免依赖外部 CSS/图片
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>%s</title>
</head>
<body style="margin:0;padding:0;background:#f4f5f7;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,'PingFang SC','Hiragino Sans GB','Microsoft YaHei',sans-serif;color:#1d1d1f;">
  <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="background:#f4f5f7;padding:32px 12px;">
    <tr>
      <td align="center">
        <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="max-width:520px;background:#ffffff;border-radius:16px;overflow:hidden;border:1px solid #e5e7eb;">
          <tr>
            <td style="padding:28px 28px 18px 28px;background:linear-gradient(135deg,#0a84ff 0%%,#409cff 100%%);">
              <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0">
                <tr>
                  <td style="color:#ffffff;font-size:13px;letter-spacing:0.04em;opacity:0.92;">Ensis Cloud</td>
                </tr>
                <tr>
                  <td style="padding-top:8px;color:#ffffff;font-size:22px;font-weight:700;letter-spacing:-0.02em;">%s</td>
                </tr>
              </table>
            </td>
          </tr>
          <tr>
            <td style="padding:28px;">
              <p style="margin:0 0 18px 0;font-size:15px;line-height:1.65;color:#3a3a3c;">%s</p>
              <table role="presentation" width="100%%" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 20px 0;">
                <tr>
                  <td align="center" style="background:#f2f7ff;border:1px solid #d6e6ff;border-radius:12px;padding:22px 16px;">
                    <div style="font-size:12px;color:#6b7280;margin-bottom:8px;">%s</div>
                    <div style="font-size:36px;font-weight:750;letter-spacing:0.28em;color:#0a84ff;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;">%s</div>
                  </td>
                </tr>
              </table>
              <p style="margin:0 0 10px 0;font-size:13px;line-height:1.6;color:#6b7280;">验证码 <strong style="color:#1d1d1f;">%d 分钟</strong> 内有效，请勿泄露给他人。</p>
              <p style="margin:0;font-size:13px;line-height:1.6;color:#6b7280;">如非本人操作，请忽略本邮件，你的账户仍然安全。</p>
            </td>
          </tr>
          <tr>
            <td style="padding:16px 28px 24px 28px;border-top:1px solid #eef0f3;">
              <p style="margin:0;font-size:12px;line-height:1.55;color:#9ca3af;">
                — Ensis 团队<br />
                <a href="https://www.niuma007.com" style="color:#0a84ff;text-decoration:none;">www.niuma007.com</a>
              </p>
            </td>
          </tr>
        </table>
      </td>
    </tr>
  </table>
</body>
</html>`, safeTitle, safeTitle, safeLead, safeAction, safeCode, ttlMin)
}
