package db

import (
	"database/sql"
	_ "embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Open opens (or creates) a SQLite database at path and applies the schema.
// It enables foreign keys and WAL mode.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, err
	}
	if _, err := d.Exec(schemaSQL); err != nil {
		d.Close()
		return nil, fmt.Errorf("apply schema: %w", err)
	}
	return d, nil
}

// SeedIfEmpty inserts a default set of categories, settings and a couple of
// starter accounts when the database is brand new. It is a no-op if any
// non-trivial data already exists.
func SeedIfEmpty(d *sql.DB) error {
	var n int
	if err := d.QueryRow(`SELECT COUNT(*) FROM accounts`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	settings := map[string]string{
		"company_name": "我的公司",
		"fiscal_year":  "2026",
	}
	for k, v := range settings {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO settings(key, value) VALUES(?,?)`, k, v); err != nil {
			return err
		}
	}

	type catRow struct {
		name  string
		group string
	}
	cats := []catRow{
		{"廣告費", "expense"},
		{"辦公用品", "expense"},
		{"產品研發", "expense"},
		{"保險", "expense"},
		{"薪資", "expense"},
		{"修繕費用", "expense"},
		{"外包費用", "expense"},
		{"交通", "expense"},
		{"餐飲", "expense"},
		{"公用事業", "expense"},
		{"租用費用", "expense"},
		{"貸款費用", "expense"},
		{"稅", "expense"},
		{"郵費", "expense"},
		{"銀行費用", "expense"},
		{"其他支出", "expense"},
		{"商品1成本", "cost"},
		{"商品2成本", "cost"},
		{"商品1收入", "income"},
		{"商品2收入", "income"},
		{"其他收入", "income"},
		{"實收資本", "equity"},
		{"轉帳沖銷", "other"},
		{"前期結轉", "other"},
	}
	for i, c := range cats {
		if _, err := tx.Exec(
			`INSERT INTO categories(name, group_name, sort_order) VALUES(?,?,?)`,
			c.name, c.group, i,
		); err != nil {
			return err
		}
	}

	accs := []struct {
		name string
		kind string
	}{
		{"銀行帳戶", "asset"},
		{"零用金", "asset"},
		{"信用卡", "liability"},
	}
	for i, a := range accs {
		if _, err := tx.Exec(
			`INSERT INTO accounts(name, kind, sort_order) VALUES(?,?,?)`,
			a.name, a.kind, i,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
