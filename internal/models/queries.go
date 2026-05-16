package models

import (
	"database/sql"
	"fmt"
	"strings"
)

// ----- Settings -----

func GetSetting(d *sql.DB, key string) (string, error) {
	var v string
	err := d.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return v, err
}

func SetSetting(d *sql.DB, key, value string) error {
	_, err := d.Exec(
		`INSERT INTO settings(key,value) VALUES(?,?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

func AllSettings(d *sql.DB) (map[string]string, error) {
	rows, err := d.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]string{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		m[k] = v
	}
	return m, rows.Err()
}

// ----- Accounts -----

func ListAccounts(d *sql.DB, activeOnly bool) ([]Account, error) {
	q := `SELECT id, name, kind, active, sort_order FROM accounts`
	if activeOnly {
		q += ` WHERE active=1`
	}
	q += ` ORDER BY kind DESC, sort_order, id`
	rows, err := d.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		var a Account
		var active int
		if err := rows.Scan(&a.ID, &a.Name, &a.Kind, &active, &a.SortOrder); err != nil {
			return nil, err
		}
		a.Active = active == 1
		out = append(out, a)
	}
	return out, rows.Err()
}

func GetAccount(d *sql.DB, id int64) (*Account, error) {
	var a Account
	var active int
	err := d.QueryRow(
		`SELECT id, name, kind, active, sort_order FROM accounts WHERE id=?`, id,
	).Scan(&a.ID, &a.Name, &a.Kind, &active, &a.SortOrder)
	if err != nil {
		return nil, err
	}
	a.Active = active == 1
	return &a, nil
}

func CreateAccount(d *sql.DB, a *Account) (int64, error) {
	res, err := d.Exec(
		`INSERT INTO accounts(name, kind, active, sort_order) VALUES(?,?,?,?)`,
		a.Name, a.Kind, boolInt(a.Active), a.SortOrder,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateAccount(d *sql.DB, a *Account) error {
	_, err := d.Exec(
		`UPDATE accounts SET name=?, kind=?, active=?, sort_order=? WHERE id=?`,
		a.Name, a.Kind, boolInt(a.Active), a.SortOrder, a.ID,
	)
	return err
}

func DeleteAccount(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM accounts WHERE id=?`, id)
	return err
}

// AccountBalance computes the current balance (in cents) of the given account
// as: sum(amount when to_account=id) - sum(amount when from_account=id).
func AccountBalance(d *sql.DB, id int64) (int64, error) {
	var in, out sql.NullInt64
	if err := d.QueryRow(
		`SELECT COALESCE(SUM(amount_cents),0) FROM transactions WHERE to_account_id=?`, id,
	).Scan(&in); err != nil {
		return 0, err
	}
	if err := d.QueryRow(
		`SELECT COALESCE(SUM(amount_cents),0) FROM transactions WHERE from_account_id=?`, id,
	).Scan(&out); err != nil {
		return 0, err
	}
	return in.Int64 - out.Int64, nil
}

// AccountBalances returns a map[accountID]balance for all accounts.
func AccountBalances(d *sql.DB) (map[int64]int64, error) {
	out := map[int64]int64{}
	rows, err := d.Query(`SELECT id FROM accounts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	for _, id := range ids {
		b, err := AccountBalance(d, id)
		if err != nil {
			return nil, err
		}
		out[id] = b
	}
	return out, nil
}

// ----- Categories -----

func ListCategories(d *sql.DB) ([]Category, error) {
	rows, err := d.Query(
		`SELECT id, name, group_name, sort_order FROM categories ORDER BY group_name, sort_order, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Category
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.Name, &c.Group, &c.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func GetCategory(d *sql.DB, id int64) (*Category, error) {
	var c Category
	err := d.QueryRow(
		`SELECT id, name, group_name, sort_order FROM categories WHERE id=?`, id,
	).Scan(&c.ID, &c.Name, &c.Group, &c.SortOrder)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func CreateCategory(d *sql.DB, c *Category) (int64, error) {
	res, err := d.Exec(
		`INSERT INTO categories(name, group_name, sort_order) VALUES(?,?,?)`,
		c.Name, c.Group, c.SortOrder,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateCategory(d *sql.DB, c *Category) error {
	_, err := d.Exec(
		`UPDATE categories SET name=?, group_name=?, sort_order=? WHERE id=?`,
		c.Name, c.Group, c.SortOrder, c.ID,
	)
	return err
}

func DeleteCategory(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM categories WHERE id=?`, id)
	return err
}

// ----- Projects -----

func ListProjects(d *sql.DB) ([]Project, error) {
	rows, err := d.Query(
		`SELECT id, name, start_date, end_date, note FROM projects ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.StartDate, &p.EndDate, &p.Note); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func GetProject(d *sql.DB, id int64) (*Project, error) {
	var p Project
	err := d.QueryRow(
		`SELECT id, name, start_date, end_date, note FROM projects WHERE id=?`, id,
	).Scan(&p.ID, &p.Name, &p.StartDate, &p.EndDate, &p.Note)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreateProject(d *sql.DB, p *Project) (int64, error) {
	res, err := d.Exec(
		`INSERT INTO projects(name, start_date, end_date, note) VALUES(?,?,?,?)`,
		p.Name, nullableDate(p.StartDate), nullableDate(p.EndDate), p.Note,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateProject(d *sql.DB, p *Project) error {
	_, err := d.Exec(
		`UPDATE projects SET name=?, start_date=?, end_date=?, note=? WHERE id=?`,
		p.Name, nullableDate(p.StartDate), nullableDate(p.EndDate), p.Note, p.ID,
	)
	return err
}

func DeleteProject(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM projects WHERE id=?`, id)
	return err
}

// ----- Transactions -----

type TxFilter struct {
	YearMonth   string // "" or "YYYY-MM"
	Year        string // "" or "YYYY"
	CategoryID  int64
	ProjectID   int64
	AccountID   int64
	SearchText  string
	Limit       int
	Offset      int
}

func ListTransactions(d *sql.DB, f TxFilter) ([]Transaction, int, error) {
	var (
		where []string
		args  []any
	)
	if f.YearMonth != "" {
		where = append(where, `substr(t.tx_date,1,7)=?`)
		args = append(args, f.YearMonth)
	} else if f.Year != "" {
		where = append(where, `substr(t.tx_date,1,4)=?`)
		args = append(args, f.Year)
	}
	if f.CategoryID > 0 {
		where = append(where, `t.category_id=?`)
		args = append(args, f.CategoryID)
	}
	if f.ProjectID > 0 {
		where = append(where, `t.project_id=?`)
		args = append(args, f.ProjectID)
	}
	if f.AccountID > 0 {
		where = append(where, `(t.from_account_id=? OR t.to_account_id=?)`)
		args = append(args, f.AccountID, f.AccountID)
	}
	if f.SearchText != "" {
		where = append(where, `(t.description LIKE ? OR t.note LIKE ? OR t.code LIKE ?)`)
		like := "%" + f.SearchText + "%"
		args = append(args, like, like, like)
	}
	clause := ""
	if len(where) > 0 {
		clause = " WHERE " + strings.Join(where, " AND ")
	}

	// Count
	var total int
	countQ := `SELECT COUNT(*) FROM transactions t` + clause
	if err := d.QueryRow(countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count: %w", err)
	}

	q := `
        SELECT t.id, t.code, t.tx_date, t.description, t.category_id, t.amount_cents,
               t.from_account_id, t.to_account_id, t.project_id, t.note,
               COALESCE(c.name,''), COALESCE(fa.name,''), COALESCE(ta.name,''), COALESCE(p.name,'')
        FROM transactions t
        LEFT JOIN categories c ON c.id=t.category_id
        LEFT JOIN accounts   fa ON fa.id=t.from_account_id
        LEFT JOIN accounts   ta ON ta.id=t.to_account_id
        LEFT JOIN projects   p  ON p.id=t.project_id
    ` + clause + ` ORDER BY t.tx_date DESC, t.id DESC`
	if f.Limit > 0 {
		q += fmt.Sprintf(" LIMIT %d OFFSET %d", f.Limit, f.Offset)
	}
	rows, err := d.Query(q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(
			&t.ID, &t.Code, &t.Date, &t.Description, &t.CategoryID, &t.AmountCents,
			&t.FromAccountID, &t.ToAccountID, &t.ProjectID, &t.Note,
			&t.CategoryName, &t.FromAccountName, &t.ToAccountName, &t.ProjectName,
		); err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func GetTransaction(d *sql.DB, id int64) (*Transaction, error) {
	var t Transaction
	err := d.QueryRow(`
        SELECT t.id, t.code, t.tx_date, t.description, t.category_id, t.amount_cents,
               t.from_account_id, t.to_account_id, t.project_id, t.note
        FROM transactions t WHERE t.id=?`, id,
	).Scan(
		&t.ID, &t.Code, &t.Date, &t.Description, &t.CategoryID, &t.AmountCents,
		&t.FromAccountID, &t.ToAccountID, &t.ProjectID, &t.Note,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func CreateTransaction(d *sql.DB, t *Transaction) (int64, error) {
	res, err := d.Exec(`
        INSERT INTO transactions(
            code, tx_date, description, category_id, amount_cents,
            from_account_id, to_account_id, project_id, note
        ) VALUES(?,?,?,?,?,?,?,?,?)`,
		t.Code, t.Date, t.Description, t.CategoryID, t.AmountCents,
		t.FromAccountID, t.ToAccountID, t.ProjectID, t.Note,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateTransaction(d *sql.DB, t *Transaction) error {
	_, err := d.Exec(`
        UPDATE transactions SET
            tx_date=?, description=?, category_id=?, amount_cents=?,
            from_account_id=?, to_account_id=?, project_id=?, note=?,
            updated_at=datetime('now')
        WHERE id=?`,
		t.Date, t.Description, t.CategoryID, t.AmountCents,
		t.FromAccountID, t.ToAccountID, t.ProjectID, t.Note, t.ID,
	)
	return err
}

func DeleteTransaction(d *sql.DB, id int64) error {
	_, err := d.Exec(`DELETE FROM transactions WHERE id=?`, id)
	return err
}

// ----- Helpers -----

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableDate(s sql.NullString) any {
	if !s.Valid || s.String == "" {
		return nil
	}
	return s.String
}

// NullStringFrom returns a sql.NullString that is invalid for "" and valid otherwise.
func NullStringFrom(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// NullInt64From returns a sql.NullInt64 that is invalid for 0 and valid otherwise.
func NullInt64From(n int64) sql.NullInt64 {
	if n == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: n, Valid: true}
}
