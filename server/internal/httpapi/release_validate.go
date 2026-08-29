package httpapi

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

var (
	sha256HexRe = regexp.MustCompile(`^[a-f0-9]{64}$`)
	productRe   = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
)

const (
	maxTitleRunes  = 200
	maxNotesRunes  = 20000
	maxDownloadURL = 2048
	defaultProduct = "niuma"
	defaultChannel = "stable"
)

// allowedDownloadHosts 非空时，download_url host 必须命中（防运营误填内网/任意站）。
// 空切片：仅拒绝非 HTTPS、凭据、IP 字面量与保留/私网地址。
func validateDownloadURL(raw string, allowedHosts []string) error {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxDownloadURL {
		return fmt.Errorf("invalid_download_url")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid_download_url")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("download_url_https_required")
	}
	if u.User != nil {
		return fmt.Errorf("download_url_credentials_forbidden")
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("invalid_download_url")
	}
	// 禁止指向本机与链路本地名
	switch host {
	case "localhost", "localhost.", "metadata.google.internal":
		return fmt.Errorf("download_url_host_forbidden")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
			ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
			return fmt.Errorf("download_url_host_forbidden")
		}
		// 公网 IP 字面量也不鼓励；若未配置 allowlist 则拒绝 IP，逼运营用域名
		if len(allowedHosts) == 0 {
			return fmt.Errorf("download_url_host_forbidden")
		}
	}
	if len(allowedHosts) > 0 {
		ok := false
		for _, h := range allowedHosts {
			h = strings.ToLower(strings.TrimSpace(h))
			if h == "" {
				continue
			}
			if host == h || strings.HasSuffix(host, "."+h) {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("download_url_host_not_allowed")
		}
	}
	return nil
}

func normalizeReleaseDims(product, channel, platform, arch string) (string, string, string, string, error) {
	product = strings.ToLower(strings.TrimSpace(product))
	if product == "" {
		product = defaultProduct
	}
	if !productRe.MatchString(product) {
		return "", "", "", "", fmt.Errorf("invalid_product")
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	if channel == "" {
		channel = defaultChannel
	}
	if channel != "stable" && channel != "beta" {
		return "", "", "", "", fmt.Errorf("invalid_channel")
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	switch platform {
	case "windows", "linux", "macos":
	default:
		return "", "", "", "", fmt.Errorf("invalid_platform")
	}
	arch = strings.ToLower(strings.TrimSpace(arch))
	switch arch {
	case "x64", "arm64":
	default:
		return "", "", "", "", fmt.Errorf("invalid_arch")
	}
	return product, channel, platform, arch, nil
}

func validateReleaseFields(r *store.AppRelease, allowedHosts []string) error {
	if err := validateReleaseMeta(r); err != nil {
		return err
	}
	if err := validateDownloadURL(r.DownloadURL, allowedHosts); err != nil {
		return err
	}
	r.DownloadURL = strings.TrimSpace(r.DownloadURL)
	return nil
}

// validateReleaseMeta 校验发布字段，不含 download_url（本机安装包由 PublicURL 生成）。
func validateReleaseMeta(r *store.AppRelease) error {
	if r == nil {
		return fmt.Errorf("invalid_release")
	}
	var err error
	r.Product, r.Channel, r.Platform, r.Arch, err = normalizeReleaseDims(r.Product, r.Channel, r.Platform, r.Arch)
	if err != nil {
		return err
	}
	v, err := store.ParseSemVer(r.Version)
	if err != nil {
		return fmt.Errorf("invalid_version")
	}
	r.Version = v.String()
	r.Title = strings.TrimSpace(r.Title)
	if r.Title == "" || utf8.RuneCountInString(r.Title) > maxTitleRunes {
		return fmt.Errorf("invalid_title")
	}
	if utf8.RuneCountInString(r.NotesMd) > maxNotesRunes {
		return fmt.Errorf("notes_too_large")
	}
	r.SHA256 = strings.ToLower(strings.TrimSpace(r.SHA256))
	if !sha256HexRe.MatchString(r.SHA256) {
		return fmt.Errorf("invalid_sha256")
	}
	if r.FileSize < 0 {
		return fmt.Errorf("invalid_file_size")
	}
	r.MinSupportedVersion = strings.TrimSpace(r.MinSupportedVersion)
	if r.MinSupportedVersion != "" {
		mv, err := store.ParseSemVer(r.MinSupportedVersion)
		if err != nil {
			return fmt.Errorf("invalid_min_supported_version")
		}
		r.MinSupportedVersion = mv.String()
	}
	switch r.Status {
	case "draft", "published", "yanked":
	default:
		return fmt.Errorf("invalid_status")
	}
	return nil
}

func computeForceUpdate(current string, latest *store.AppRelease) bool {
	if latest == nil {
		return false
	}
	if latest.ForceUpdate {
		return true
	}
	if latest.MinSupportedVersion == "" {
		return false
	}
	cmp, err := store.CompareSemVerStrings(current, latest.MinSupportedVersion)
	if err != nil {
		return false
	}
	return cmp < 0
}
