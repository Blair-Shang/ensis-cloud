package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Store struct {
	db     *sql.DB
	driver string
}

func New(db *sql.DB, driver string) *Store {
	return &Store{db: db, driver: strings.ToLower(driver)}
}

const (
	defaultPageSize = 20
	maxPageSize     = 50
)

// clampPage 运营列表分页：默认 20，上限 50，避免一次拉全表。
func clampPage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func appendEq(where *[]string, args *[]any, col, val string) {
	val = strings.TrimSpace(val)
	if val == "" {
		return
	}
	*where = append(*where, col+" = ?")
	*args = append(*args, val)
}

func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

func nowRFC3339() string {
	return formatTime(nowUTC())
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// dbTime 兼容 MySQL DATETIME 与 SQLite TEXT 时间扫描。
type dbTime struct {
	Time  time.Time
	Valid bool
}

func (t *dbTime) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		t.Valid = false
		t.Time = time.Time{}
		return nil
	case time.Time:
		t.Time = v.UTC()
		t.Valid = !t.Time.IsZero()
		return nil
	case []byte:
		return t.parse(string(v))
	case string:
		return t.parse(v)
	default:
		return fmt.Errorf("cannot scan %T into dbTime", src)
	}
}

func (t *dbTime) parse(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		t.Valid = false
		t.Time = time.Time{}
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		// SQLite TEXT 写入 time.Time 时常见为 Go 默认 String()
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if parsed, err := time.ParseInLocation(layout, s, time.UTC); err == nil {
			t.Time = parsed.UTC()
			t.Valid = true
			return nil
		}
	}
	return fmt.Errorf("invalid time %q", s)
}

func (t dbTime) String() string {
	if !t.Valid {
		return ""
	}
	return formatTime(t.Time)
}

func (t dbTime) Ptr() *string {
	if !t.Valid {
		return nil
	}
	s := formatTime(t.Time)
	return &s
}

// toDBTime 将 API 层 RFC3339 字符串转为可写入 DATETIME 的值。
func toDBTime(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC()
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC()
	}
	return s
}

func toDBNullTime(p *string) any {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil
	}
	return toDBTime(*p)
}
