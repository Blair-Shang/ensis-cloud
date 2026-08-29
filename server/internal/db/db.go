package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "modernc.org/sqlite"
)

func Open(driver, dsn string) (*sql.DB, error) {
	var sqlDriver string
	switch strings.ToLower(driver) {
	case "sqlite":
		sqlDriver = "sqlite"
		if dir := filepath.Dir(dsn); dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				return nil, fmt.Errorf("mkdir sqlite dir: %w", err)
			}
		}
		if !strings.Contains(dsn, "?") {
			dsn += "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
		}
	case "mysql":
		sqlDriver = "mysql"
		if err := ensureMySQLDatabase(dsn); err != nil {
			return nil, err
		}
		dsn = withMySQLParams(dsn)
	default:
		return nil, fmt.Errorf("unsupported driver %q", driver)
	}

	db, err := sql.Open(sqlDriver, dsn)
	if err != nil {
		return nil, err
	}
	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
	}
	if err = db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping %s: %w", driver, err)
	}
	if err = migrate(db, strings.ToLower(driver)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func withMySQLParams(dsn string) string {
	if strings.Contains(dsn, "parseTime=") {
		return dsn
	}
	if strings.Contains(dsn, "?") {
		return dsn + "&parseTime=true&charset=utf8mb4&loc=UTC"
	}
	return dsn + "?parseTime=true&charset=utf8mb4&loc=UTC"
}

func splitMySQLDSN(dsn string) (userHost, dbName, query string, ok bool) {
	i := strings.Index(dsn, "@tcp(")
	if i < 0 {
		return "", "", "", false
	}
	j := strings.Index(dsn[i:], ")/")
	if j < 0 {
		return "", "", "", false
	}
	// j 指向 ')'；userHost 含至 ')'，rest 为库名与 query
	paren := i + j
	userHost = dsn[:paren+1]
	rest := dsn[paren+2:]
	if q := strings.Index(rest, "?"); q >= 0 {
		dbName = rest[:q]
		query = rest[q:]
	} else {
		dbName = rest
	}
	return userHost, strings.TrimSpace(dbName), query, true
}

// ensureMySQLDatabase 在目标库不存在时自动 CREATE DATABASE。
func ensureMySQLDatabase(dsn string) error {
	userHost, dbName, query, ok := splitMySQLDSN(dsn)
	if !ok {
		return fmt.Errorf("invalid mysql DSN (want user:pass@tcp(host:port)/dbname)")
	}
	if dbName == "" {
		return fmt.Errorf("mysql DSN missing database name")
	}
	adminDSN := withMySQLParams(userHost + "/" + query)
	if query == "" {
		adminDSN = withMySQLParams(userHost + "/")
	}

	db, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return fmt.Errorf("open mysql admin: %w", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql admin: %w", err)
	}
	safeName := strings.ReplaceAll(dbName, "`", "")
	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS `" + safeName + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
	if err != nil {
		return fmt.Errorf("create database %s: %w", dbName, err)
	}
	return nil
}
