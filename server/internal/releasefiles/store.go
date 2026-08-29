package releasefiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	dimRe  = regexp.MustCompile(`^[a-z][a-z0-9._-]{0,63}$`)
	verRe  = regexp.MustCompile(`^[0-9]+(\.[0-9]+){1,3}([.-][A-Za-z0-9]+)*$`)
	nameRe = regexp.MustCompile(`(?i)^[A-Za-z0-9][A-Za-z0-9._-]{0,190}\.(exe|msi|pkg|dmg|deb|rpm|run)$`)
)

// Dim 安装包落盘维度。
type Dim struct {
	Product  string
	Channel  string
	Platform string
	Arch     string
	Version  string
	Name     string
}

// Store 把安装包存在本机 data 目录，供国内用户与桌面更新下载。
type Store struct {
	root string
}

// New 创建落盘存储；root 一般为 {data_dir}/releases。
func New(root string) *Store {
	return &Store{root: filepath.Clean(root)}
}

// Root 返回安装包根目录。
func (s *Store) Root() string {
	return s.root
}

// Normalize 校验并规范化维度与文件名。
func Normalize(d Dim) (Dim, error) {
	d.Product = strings.ToLower(strings.TrimSpace(d.Product))
	d.Channel = strings.ToLower(strings.TrimSpace(d.Channel))
	d.Platform = strings.ToLower(strings.TrimSpace(d.Platform))
	d.Arch = strings.ToLower(strings.TrimSpace(d.Arch))
	d.Version = strings.TrimSpace(d.Version)
	name := strings.TrimSpace(d.Name)
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return Dim{}, fmt.Errorf("invalid_installer_name")
	}
	d.Name = filepath.Base(name)
	if !dimRe.MatchString(d.Product) || !dimRe.MatchString(d.Channel) ||
		!dimRe.MatchString(d.Platform) || !dimRe.MatchString(d.Arch) {
		return Dim{}, fmt.Errorf("invalid_release_path")
	}
	if !verRe.MatchString(d.Version) {
		return Dim{}, fmt.Errorf("invalid_version")
	}
	if !nameRe.MatchString(d.Name) {
		return Dim{}, fmt.Errorf("invalid_installer_name")
	}
	return d, nil
}

// FilePath 返回本机绝对路径（须先 Normalize）。
func (s *Store) FilePath(d Dim) string {
	return filepath.Join(s.root, d.Product, d.Channel, d.Platform, d.Arch, d.Version, d.Name)
}

// Exists 判断安装包是否已落盘。
func (s *Store) Exists(d Dim) bool {
	st, err := os.Stat(s.FilePath(d))
	return err == nil && st.Mode().IsRegular()
}

// Save 流式写入安装包，边写边算 SHA-256；expectSHA 非空则必须一致。
func (s *Store) Save(d Dim, r io.Reader, maxBytes int64, expectSHA string) (size int64, sum string, err error) {
	d, err = Normalize(d)
	if err != nil {
		return 0, "", err
	}
	if maxBytes <= 0 {
		maxBytes = 2 << 30
	}
	dir := filepath.Dir(s.FilePath(d))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, "", err
	}
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return 0, "", err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if err != nil {
			_ = os.Remove(tmpName)
		}
	}()

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), io.LimitReader(r, maxBytes+1))
	if err != nil {
		return 0, "", err
	}
	if n > maxBytes {
		return 0, "", fmt.Errorf("installer_too_large")
	}
	sum = hex.EncodeToString(h.Sum(nil))
	expectSHA = strings.ToLower(strings.TrimSpace(expectSHA))
	if expectSHA != "" && expectSHA != sum {
		return 0, "", fmt.Errorf("hash_mismatch")
	}
	if err = tmp.Close(); err != nil {
		return 0, "", err
	}
	if err = replaceFile(tmpName, s.FilePath(d)); err != nil {
		return 0, "", err
	}
	return n, sum, nil
}

// replaceFile 用临时文件覆盖目标；Windows 上 Rename 不能覆盖已存在文件。
func replaceFile(tmp, final string) error {
	if err := os.Rename(tmp, final); err == nil {
		return nil
	}
	if err := os.Remove(final); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(tmp, final)
}

// Resolve 按维度定位已落盘安装包；Name 为空时该版本目录须恰好一个合法文件。
func (s *Store) Resolve(d Dim) (Dim, bool) {
	d.Product = strings.ToLower(strings.TrimSpace(d.Product))
	d.Channel = strings.ToLower(strings.TrimSpace(d.Channel))
	d.Platform = strings.ToLower(strings.TrimSpace(d.Platform))
	d.Arch = strings.ToLower(strings.TrimSpace(d.Arch))
	d.Version = strings.TrimSpace(d.Version)
	d.Name = filepath.Base(strings.TrimSpace(d.Name))
	if d.Name != "" && d.Name != "." {
		nd, err := Normalize(d)
		if err != nil {
			return Dim{}, false
		}
		return nd, s.Exists(nd)
	}
	dir := filepath.Join(s.root, d.Product, d.Channel, d.Platform, d.Arch, d.Version)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return Dim{}, false
	}
	var found Dim
	n := 0
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		cand := d
		cand.Name = e.Name()
		nd, nerr := Normalize(cand)
		if nerr != nil {
			continue
		}
		n++
		found = nd
		if n > 1 {
			return Dim{}, false
		}
	}
	if n != 1 {
		return Dim{}, false
	}
	return found, true
}

// PublicURL 生成对外下载地址（桌面 / 官网使用）。
func PublicURL(publicBase string, d Dim) string {
	base := strings.TrimRight(strings.TrimSpace(publicBase), "/")
	segs := []string{
		"api", "v1", "updates", "files",
		d.Product, d.Channel, d.Platform, d.Arch, d.Version, d.Name,
	}
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return base + "/" + strings.Join(segs, "/")
}
