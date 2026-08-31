package llm

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrUnsafeUpstream 表示地址不是可出站的公网 HTTPS。
var ErrUnsafeUpstream = errors.New("unsafe upstream")

var cgnat = net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

// ValidateUpstreamURL 要求 https、无用户信息、主机解析后全部为公网地址。
func ValidateUpstreamURL(raw string) error {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return ErrUnsafeUpstream
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return ErrUnsafeUpstream
	}
	if u.User != nil {
		return ErrUnsafeUpstream
	}
	host := strings.TrimSpace(u.Hostname())
	if host == "" || isBlockedHostname(host) {
		return ErrUnsafeUpstream
	}
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return ErrUnsafeUpstream
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return ErrUnsafeUpstream
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return ErrUnsafeUpstream
		}
	}
	return nil
}

// NewSafeClient 禁止跟随跳转，拨号前再次核对目标 IP。
func NewSafeClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultChatTimeout
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("redirect forbidden")
		},
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          32,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				host, port, err := net.SplitHostPort(addr)
				if err != nil {
					return nil, ErrUnsafeUpstream
				}
				ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
				if err != nil || len(ips) == 0 {
					return nil, ErrUnsafeUpstream
				}
				var last error
				for _, ipa := range ips {
					if !isPublicIP(ipa.IP) {
						last = ErrUnsafeUpstream
						continue
					}
					c, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ipa.IP.String(), port))
					if dialErr == nil {
						return c, nil
					}
					last = dialErr
				}
				if last == nil {
					last = ErrUnsafeUpstream
				}
				return nil, last
			},
		},
	}
}

// SanitizeDialError 去掉拨号错误里的内网地址，避免探测接口泄露拓扑。
func SanitizeDialError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "upstream timeout"
	}
	if errors.Is(err, ErrUnsafeUpstream) {
		return ErrUnsafeUpstream.Error()
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") {
		return "upstream timeout"
	}
	if strings.Contains(msg, "redirect") {
		return "redirect forbidden"
	}
	return "upstream unreachable"
}

func isBlockedHostname(host string) bool {
	h := strings.ToLower(strings.TrimSuffix(host, "."))
	switch h {
	case "localhost", "localhost.localdomain", "metadata.google.internal":
		return true
	}
	if strings.HasSuffix(h, ".localhost") || strings.HasSuffix(h, ".local") || strings.HasSuffix(h, ".internal") {
		return true
	}
	return false
}

func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() {
		return false
	}
	if v4 := ip.To4(); v4 != nil && cgnat.Contains(v4) {
		return false
	}
	return true
}
