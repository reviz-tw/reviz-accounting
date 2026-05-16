package handlers

import (
	"net/http"

	"github.com/hcchien/reviz-accounting/internal/auth"
	"github.com/hcchien/reviz-accounting/internal/models"
)

func (s *Server) categoriesList(w http.ResponseWriter, r *http.Request) {
	cats, err := models.ListCategories(s.DB)
	if err != nil {
		s.error500(w, err)
		return
	}
	grouped := map[string][]models.Category{}
	for _, c := range cats {
		grouped[c.Group] = append(grouped[c.Group], c)
	}
	type colSpec struct {
		Group string
		Label string
		Color string
	}
	s.render(w, r, "categories.html", map[string]any{
		"Title":   "分類",
		"Crumbs":  []string{"分類"},
		"Groups":  []string{"income", "cost", "expense", "equity", "other"},
		"Grouped": grouped,
		"ColumnSpecs": []colSpec{
			{"income", "收入", "var(--success-500)"},
			{"cost", "成本", "var(--warning-500)"},
			{"expense", "費用", "var(--danger-500)"},
		},
		"Active": "categories",
	})
}

func (s *Server) categoryCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.error500(w, err)
		return
	}
	name := r.FormValue("name")
	group := r.FormValue("group_name")
	if name == "" || group == "" {
		http.Error(w, "name and group required", http.StatusBadRequest)
		return
	}
	_, err := models.CreateCategory(s.DB, &models.Category{Name: name, Group: group})
	if err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (s *Server) categoryUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if err := r.ParseForm(); err != nil {
		s.error500(w, err)
		return
	}
	c, err := models.GetCategory(s.DB, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if v := r.FormValue("name"); v != "" {
		c.Name = v
	}
	if v := r.FormValue("group_name"); v != "" {
		c.Group = v
	}
	if err := models.UpdateCategory(s.DB, c); err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}

func (s *Server) categoryDelete(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if err := models.DeleteCategory(s.DB, id); err != nil {
		auth.WriteAlertRedirect(w, r, http.StatusConflict,
			"刪除失敗：此分類仍有交易記錄。", "/categories")
		return
	}
	http.Redirect(w, r, "/categories", http.StatusSeeOther)
}
