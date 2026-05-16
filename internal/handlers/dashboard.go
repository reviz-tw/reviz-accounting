package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/hcchien/reviz-accounting/internal/models"
)

type dashKPI struct {
	Label       string
	DotClass    string
	ValueMain   string // pre-decimal portion (e.g. "10,000")
	ValueFrac   string // ".00" portion
	ValueClass  string // "is-pos" | "is-neg" | ""
	ValueColor  string // optional inline color override for the main number
	Sub         template.HTML
}

type dashCatSlice struct {
	Name string
	Pct  int    // 0..100
	Disp string // formatted value
}

type dashboardData struct {
	Year         int
	Years        []int
	NetWorth     int64
	AssetTotal   int64
	LiabTotal    int64
	IncomeYTD    int64
	CostYTD      int64
	ExpenseYTD   int64
	GrossYTD     int64
	NetProfitYTD int64
	TxCount      int

	KPIs        []dashKPI
	MonthlyChart template.HTML
	NetChart     template.HTML

	TopExpenses []dashCatSlice
	TopIncome   []dashCatSlice
	ExpenseTotal int64
	IncomeTotal  int64
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	year, _ := strconv.Atoi(q.Get("year"))
	if year == 0 {
		fy, _ := models.GetSetting(s.DB, "fiscal_year")
		year, _ = strconv.Atoi(fy)
	}
	if year == 0 {
		year = time.Now().Year()
	}
	pnl, err := models.ComputePnL(s.DB, year, 0)
	if err != nil {
		s.error500(w, err)
		return
	}
	bal, err := models.AccountBalances(s.DB)
	if err != nil {
		s.error500(w, err)
		return
	}
	accs, err := models.ListAccounts(s.DB, false)
	if err != nil {
		s.error500(w, err)
		return
	}
	var aTot, lTot int64
	for _, a := range accs {
		if a.Kind == "asset" {
			aTot += bal[a.ID]
		} else {
			lTot += bal[a.ID]
		}
	}

	incomeSeries := append([]int64(nil), pnl.IncomeTotal[:12]...)
	costSeries := append([]int64(nil), pnl.CostTotal[:12]...)
	expSeries := append([]int64(nil), pnl.ExpenseTotal[:12]...)
	netSeries := append([]int64(nil), pnl.NetProfit[:12]...)

	// transaction count YTD
	yearStr := fmt.Sprintf("%04d", year)
	txTotal := 0
	_ = s.DB.QueryRow(
		`SELECT COUNT(*) FROM transactions WHERE substr(tx_date,1,4)=?`, yearStr,
	).Scan(&txTotal)

	// peak month for income
	peakMonth := ""
	var peakVal int64
	for i, v := range incomeSeries {
		if v > peakVal {
			peakVal = v
			peakMonth = fmt.Sprintf("%d 月", i+1)
		}
	}

	years, _ := models.AvailableYears(s.DB)
	if len(years) == 0 {
		years = []int{year}
	}

	d := dashboardData{
		Year:         year,
		Years:        years,
		NetWorth:     aTot - lTot,
		AssetTotal:   aTot,
		LiabTotal:    lTot,
		IncomeYTD:    pnl.IncomeTotal[12],
		CostYTD:      pnl.CostTotal[12],
		ExpenseYTD:   pnl.ExpenseTotal[12],
		GrossYTD:     pnl.GrossProfit[12],
		NetProfitYTD: pnl.NetProfit[12],
		TxCount:      txTotal,
		MonthlyChart: DotChartMonthlySVG(incomeSeries, costSeries, expSeries),
		NetChart:     DotChartNetSVG(netSeries),
	}

	// KPI tiles
	netClass := ""
	switch {
	case d.NetWorth < 0:
		netClass = "is-neg"
	case d.NetWorth > 0:
		netClass = "is-pos"
	}
	d.KPIs = []dashKPI{
		{
			Label: "淨值", DotClass: "lg-kpi__label-dot--net",
			ValueClass: netClass, ValueMain: splitMoney(d.NetWorth),
			ValueFrac: ".00",
			Sub: template.HTML(fmt.Sprintf(`資產 <strong>%s</strong> · 負債 <strong>%s</strong>`,
				fmtMoney(d.AssetTotal), fmtMoney(d.LiabTotal))),
		},
		{
			Label: "YTD 收入", DotClass: "lg-kpi__label-dot--income",
			ValueMain:  splitMoney(d.IncomeYTD),
			ValueFrac:  ".00",
			ValueColor: "var(--success-700)",
			Sub: template.HTML(func() string {
				if peakMonth != "" {
					return fmt.Sprintf(`%d 筆交易 · 集中於 <strong>%s</strong>`, txTotal, peakMonth)
				}
				return fmt.Sprintf(`%d 筆交易`, txTotal)
			}()),
		},
		{
			Label: "YTD 費用", DotClass: "lg-kpi__label-dot--expense",
			ValueMain:  splitMoney(d.ExpenseYTD),
			ValueFrac:  ".00",
			ValueColor: "var(--danger-700)",
			Sub:        template.HTML(fmt.Sprintf(`成本 <strong>%s</strong>`, fmtMoney(d.CostYTD))),
		},
		{
			Label: "YTD 淨利", DotClass: "lg-kpi__label-dot--net",
			ValueClass: netProfitClass(d.NetProfitYTD), ValueMain: splitMoney(d.NetProfitYTD),
			ValueFrac: ".00",
			Sub: template.HTML(fmt.Sprintf(`毛利 <strong>%s</strong>`, fmtMoney(d.GrossYTD))),
		},
	}

	// Top categories (YTD)
	d.TopExpenses, d.ExpenseTotal = topCategories(pnl.Expense, 5)
	d.TopIncome, d.IncomeTotal = topCategories(pnl.Income, 5)

	s.render(w, r, "dashboard.html", map[string]any{
		"Title":  "儀表板",
		"Crumbs": []string{"儀表板"},
		"D":      d,
		"Active": "dashboard",
	})
}

func topCategories(rows []models.PnLRow, n int) ([]dashCatSlice, int64) {
	type item struct {
		name string
		v    int64
	}
	var items []item
	var total int64
	for _, r := range rows {
		if r.YTD > 0 {
			items = append(items, item{r.CategoryName, r.YTD})
			total += r.YTD
		}
	}
	// simple insertion sort by value desc
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j-1].v < items[j].v; j-- {
			items[j-1], items[j] = items[j], items[j-1]
		}
	}
	if len(items) > n {
		items = items[:n]
	}
	out := make([]dashCatSlice, 0, len(items))
	for _, it := range items {
		pct := 0
		if total > 0 {
			pct = int(it.v * 100 / total)
		}
		out = append(out, dashCatSlice{Name: it.name, Pct: pct, Disp: fmtMoney(it.v)})
	}
	return out, total
}

func netProfitClass(v int64) string {
	switch {
	case v < 0:
		return "is-neg"
	case v > 0:
		return "is-pos"
	default:
		return ""
	}
}
