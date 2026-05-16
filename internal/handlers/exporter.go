package handlers

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/hcchien/reviz-accounting/internal/auth"
	"github.com/hcchien/reviz-accounting/internal/models"
)

const (
	sheetAccounts   = "帳戶總覽(請依照您公司需要修改)"
	sheetJournal    = "日記帳(每日記錄)"
	sheetCategories = "分類(請依照您公司需要修改)"
	sheetProjects   = "專案列表"
)

var monthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// exportMonthlyXLSX populates the Simpany template with the company's
// settings, accounts, categories, projects, and the given month's
// transactions, and streams the resulting .xlsx to the client.
//
// Route: GET /export/monthly.xlsx?month=YYYY-MM
func (s *Server) exportMonthlyXLSX(w http.ResponseWriter, r *http.Request) {
	month := r.URL.Query().Get("month")
	if month == "" {
		month = time.Now().Format("2006-01")
	}
	if !monthRe.MatchString(month) {
		auth.WriteAlertRedirect(w, r, http.StatusBadRequest,
			"月份格式錯誤，請使用 YYYY-MM", "/settings")
		return
	}

	if len(s.SimpanyTemplate) == 0 {
		s.error500(w, fmt.Errorf("simpany template not embedded"))
		return
	}
	f, err := excelize.OpenReader(bytes.NewReader(s.SimpanyTemplate))
	if err != nil {
		s.error500(w, fmt.Errorf("open template: %w", err))
		return
	}
	defer f.Close()

	if err := s.populateAccountsSheet(f); err != nil {
		s.error500(w, err)
		return
	}
	if err := s.populateCategoriesSheet(f); err != nil {
		s.error500(w, err)
		return
	}
	if err := s.populateProjectsSheet(f); err != nil {
		s.error500(w, err)
		return
	}
	if err := s.populateJournalSheet(f, month); err != nil {
		s.error500(w, err)
		return
	}

	filename := fmt.Sprintf("Reviz 帳簿 %s.xlsx", month)
	w.Header().Set("Content-Type",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename*=UTF-8''%s`, url.PathEscape(filename)))
	if err := f.Write(w); err != nil {
		// Headers already sent; log only.
		fmt.Fprintln(w, "\nWRITE ERROR:", err)
	}
}

// --- per-sheet writers ---

func (s *Server) populateAccountsSheet(f *excelize.File) error {
	company, _ := models.GetSetting(s.DB, "company_name")
	fy, _ := models.GetSetting(s.DB, "fiscal_year")
	if company != "" {
		_ = f.SetCellValue(sheetAccounts, "E5", company)
	}
	if fy != "" {
		_ = f.SetCellValue(sheetAccounts, "E6", fy)
	}

	accs, err := models.ListAccounts(s.DB, false)
	if err != nil {
		return err
	}

	const (
		assetStart = 10 // rows 10..19 (10 slots)
		liabStart  = 21 // rows 21..30 (10 slots)
		slots      = 10
	)
	// Clear D10..D19 and D21..D30 first.
	for i := 0; i < slots; i++ {
		_ = f.SetCellValue(sheetAccounts, fmt.Sprintf("D%d", assetStart+i), "")
		_ = f.SetCellValue(sheetAccounts, fmt.Sprintf("D%d", liabStart+i), "")
	}

	ai, li := 0, 0
	for _, a := range accs {
		switch a.Kind {
		case "asset":
			if ai < slots {
				_ = f.SetCellValue(sheetAccounts, fmt.Sprintf("D%d", assetStart+ai), a.Name)
				ai++
			}
		case "liability":
			if li < slots {
				_ = f.SetCellValue(sheetAccounts, fmt.Sprintf("D%d", liabStart+li), a.Name)
				li++
			}
		}
	}
	return nil
}

type catSection struct {
	startRow int
	capacity int
	label    string // shown in column A (first row) + every row of column C
}

var catSections = map[string]catSection{
	"expense": {6, 19, "費用"},
	"cost":    {25, 11, "成本"},
	"income":  {36, 7, "收入"},
	"equity":  {43, 1, "股東權益"},
	"other":   {44, 2, "其他"},
}

func (s *Server) populateCategoriesSheet(f *excelize.File) error {
	cats, err := models.ListCategories(s.DB)
	if err != nil {
		return err
	}

	// Clear rows 6..45 of A,B and column C (preserve existing C values that
	// happen to be in the right section is unnecessary — we rewrite them).
	for row := 6; row <= 45; row++ {
		_ = f.SetCellValue(sheetCategories, fmt.Sprintf("A%d", row), "")
		_ = f.SetCellValue(sheetCategories, fmt.Sprintf("B%d", row), "")
		_ = f.SetCellValue(sheetCategories, fmt.Sprintf("C%d", row), "")
	}

	grouped := map[string][]models.Category{}
	for _, c := range cats {
		grouped[c.Group] = append(grouped[c.Group], c)
	}

	for group, sec := range catSections {
		list := grouped[group]
		if len(list) > sec.capacity {
			list = list[:sec.capacity]
		}
		for i, c := range list {
			row := sec.startRow + i
			_ = f.SetCellValue(sheetCategories, fmt.Sprintf("B%d", row), c.Name)
			_ = f.SetCellValue(sheetCategories, fmt.Sprintf("C%d", row), sec.label)
			if i == 0 {
				_ = f.SetCellValue(sheetCategories, fmt.Sprintf("A%d", row), sec.label)
			}
		}
	}
	return nil
}

func (s *Server) populateProjectsSheet(f *excelize.File) error {
	projs, err := models.ListProjects(s.DB)
	if err != nil {
		return err
	}

	// Rows 3..12 are the 10 project slots in the template.
	for row := 3; row <= 12; row++ {
		for _, col := range []string{"B", "C", "D", "E"} {
			_ = f.SetCellValue(sheetProjects, fmt.Sprintf("%s%d", col, row), "")
		}
	}

	for i, p := range projs {
		if i >= 10 {
			break
		}
		row := 3 + i
		_ = f.SetCellValue(sheetProjects, fmt.Sprintf("A%d", row), float64(i+1))
		_ = f.SetCellValue(sheetProjects, fmt.Sprintf("B%d", row), p.Name)
		if p.StartDate.Valid {
			if t, err := time.Parse("2006-01-02", p.StartDate.String); err == nil {
				_ = f.SetCellValue(sheetProjects, fmt.Sprintf("C%d", row), t)
			}
		}
		if p.EndDate.Valid {
			if t, err := time.Parse("2006-01-02", p.EndDate.String); err == nil {
				_ = f.SetCellValue(sheetProjects, fmt.Sprintf("D%d", row), t)
			}
		}
		if p.Note != "" {
			_ = f.SetCellValue(sheetProjects, fmt.Sprintf("E%d", row), p.Note)
		}
	}
	return nil
}

func (s *Server) populateJournalSheet(f *excelize.File, month string) error {
	txs, _, err := models.ListTransactions(s.DB, models.TxFilter{YearMonth: month})
	if err != nil {
		return err
	}
	// Default order is DESC by date — reverse for chronological.
	sort.SliceStable(txs, func(i, j int) bool {
		if txs[i].Date != txs[j].Date {
			return txs[i].Date < txs[j].Date
		}
		return txs[i].ID < txs[j].ID
	})

	// Clear A-I for rows 4..31 (template had hard-coded codes / values) and
	// just B-I for the remaining rows (32..1006 have a formula in A that
	// auto-generates the code).
	clearCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for row := 4; row <= 31; row++ {
		for _, col := range clearCols {
			_ = f.SetCellValue(sheetJournal, fmt.Sprintf("%s%d", col, row), "")
		}
	}
	for row := 32; row <= 1006; row++ {
		for _, col := range []string{"B", "C", "D", "E", "F", "G", "H", "I"} {
			_ = f.SetCellValue(sheetJournal, fmt.Sprintf("%s%d", col, row), "")
		}
	}

	for i, t := range txs {
		if i >= 1000 {
			break
		}
		row := 4 + i
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("A%d", row), t.Code)

		if d, err := time.Parse("2006-01-02", t.Date); err == nil {
			_ = f.SetCellValue(sheetJournal, fmt.Sprintf("B%d", row), d)
		}
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("C%d", row), t.Description)
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("D%d", row), t.CategoryName)
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("E%d", row), simpanySignedAmount(&t))
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("F%d", row), t.FromAccountName)
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("G%d", row), t.ToAccountName)
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("H%d", row), t.ProjectName)
		_ = f.SetCellValue(sheetJournal, fmt.Sprintf("I%d", row), t.Note)
	}
	return nil
}

// simpanySignedAmount returns the transaction's amount expressed in Simpany's
// signed-value convention (negative for outflow, positive for inflow).
//   - income  (only to set)  -> positive
//   - expense (only from set) -> negative
//   - transfer (both set)     -> negative (from the from-account perspective)
func simpanySignedAmount(t *models.Transaction) float64 {
	v := float64(t.AmountCents) / 100.0
	switch t.Type() {
	case "income":
		return v
	default: // expense or transfer
		return -v
	}
}
