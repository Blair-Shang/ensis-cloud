package store

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer 仅支持 X.Y.Z（可选前导 v）；预发布/构建元数据 P0 不解析。
type SemVer struct {
	Major int
	Minor int
	Patch int
}

func ParseSemVer(s string) (SemVer, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimPrefix(s, "V")
	if s == "" {
		return SemVer{}, fmt.Errorf("empty version")
	}
	// 拒绝预发布等扩展，避免比较歧义
	if strings.ContainsAny(s, "-+") {
		return SemVer{}, fmt.Errorf("pre-release not supported")
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return SemVer{}, fmt.Errorf("want X.Y.Z")
	}
	var out SemVer
	for i, p := range parts {
		if p == "" || (len(p) > 1 && p[0] == '0') {
			return SemVer{}, fmt.Errorf("invalid numeric part")
		}
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return SemVer{}, fmt.Errorf("invalid numeric part")
		}
		switch i {
		case 0:
			out.Major = n
		case 1:
			out.Minor = n
		case 2:
			out.Patch = n
		}
	}
	return out, nil
}

func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// CompareSemVer 返回 -1 / 0 / 1。
func CompareSemVer(a, b SemVer) int {
	if a.Major != b.Major {
		if a.Major < b.Major {
			return -1
		}
		return 1
	}
	if a.Minor != b.Minor {
		if a.Minor < b.Minor {
			return -1
		}
		return 1
	}
	if a.Patch != b.Patch {
		if a.Patch < b.Patch {
			return -1
		}
		return 1
	}
	return 0
}

func CompareSemVerStrings(a, b string) (int, error) {
	va, err := ParseSemVer(a)
	if err != nil {
		return 0, err
	}
	vb, err := ParseSemVer(b)
	if err != nil {
		return 0, err
	}
	return CompareSemVer(va, vb), nil
}
