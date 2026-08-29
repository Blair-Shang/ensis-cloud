package db

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"
)

//go:embed migrations/mysql/*.sql
var mysqlMigrations embed.FS

//go:embed migrations/sqlite/*.sql
var sqliteMigrations embed.FS

func migrate(db *sql.DB, driver string) error {
	var root embed.FS
	var dir string
	switch driver {
	case "mysql":
		root, dir = mysqlMigrations, "migrations/mysql"
	case "sqlite":
		root, dir = sqliteMigrations, "migrations/sqlite"
	default:
		return fmt.Errorf("no migrations for driver %q", driver)
	}

	if err := ensureMigrationsTable(db, driver); err != nil {
		return err
	}

	names, err := listMigrationFiles(root, dir)
	if err != nil {
		return err
	}
	applied, err := loadAppliedVersions(db)
	if err != nil {
		return err
	}

	for _, name := range names {
		version := strings.TrimSuffix(name, ".sql")
		if applied[version] {
			continue
		}
		body, err := fs.ReadFile(root, path.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		for _, s := range splitSQL(string(body)) {
			if _, err := db.Exec(s); err != nil {
				return fmt.Errorf("exec migration %s: %w\nSQL: %s", name, err, s)
			}
		}
		if err := markApplied(db, version); err != nil {
			return err
		}
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB, driver string) error {
	var ddl string
	switch driver {
	case "mysql":
		ddl = `CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id VARCHAR(64) NOT NULL PRIMARY KEY,
			applied_at DATETIME(3) NOT NULL
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	case "sqlite":
		ddl = `CREATE TABLE IF NOT EXISTS schema_migrations (
			migration_id TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL
		)`
	default:
		return fmt.Errorf("unsupported driver %q", driver)
	}
	_, err := db.Exec(ddl)
	return err
}

func listMigrationFiles(root embed.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(root, dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names, nil
}

func loadAppliedVersions(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query(`SELECT migration_id FROM schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		out[v] = true
	}
	return out, rows.Err()
}

func markApplied(db *sql.DB, version string) error {
	_, err := db.Exec(
		`INSERT INTO schema_migrations (migration_id, applied_at) VALUES (?, ?)`,
		version, time.Now().UTC(),
	)
	return err
}

// splitSQL 按分号拆语句，忽略空段与纯注释段。
func splitSQL(src string) []string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" || strings.HasPrefix(trim, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	var stmts []string
	for _, part := range strings.Split(b.String(), ";") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		stmts = append(stmts, s)
	}
	return stmts
}
