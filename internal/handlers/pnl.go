package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/hcchien/reviz-accounting/internal/models"
)

func (s *Server) pnl(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	year, _ := strconv.Atoi(q.Get("year"))
	if year == 0 {
		fy, _ := models.GetSetting(s.DB, "fiscal_year")
		year, _ = strconv.Atoi(fy)
	}
	if year == 0 {
		year = time.Now().Year()
	}
	project := parseInt64(q.Get("project_id"))

	pnl, err := models.ComputePnL(s.DB, year, project)
	if err != nil {
		s.error500(w, err)
		return
	}
	availYears, _ := models.AvailableYears(s.DB)
	if len(availYears) == 0 {
		availYears = []int{year}
	}
	projs, _ := models.ListProjects(s.DB)

	s.render(w, r, "pnl.html", map[string]any{
		"Title":     "損益表",
		"Crumbs":    []string{"報表", "損益表"},
		"PnL":       pnl,
		"Year":      year,
		"ProjectID": project,
		"Years":     availYears,
		"Projects":  projs,
		"MonthIdx":  []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11},
		"Active":    "pnl",
	})
}
