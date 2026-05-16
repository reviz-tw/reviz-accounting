package handlers

import (
	"net/http"

	"github.com/hcchien/reviz-accounting/internal/auth"
	"github.com/hcchien/reviz-accounting/internal/models"
)

func (s *Server) accountsList(w http.ResponseWriter, r *http.Request) {
	accs, err := models.ListAccounts(s.DB, false)
	if err != nil {
		s.error500(w, err)
		return
	}
	bal, err := models.AccountBalances(s.DB)
	if err != nil {
		s.error500(w, err)
		return
	}

	type row struct {
		models.Account
		Balance int64
	}
	var assets, liabilities []row
	var assetTotal, liabilityTotal int64
	for _, a := range accs {
		r := row{Account: a, Balance: bal[a.ID]}
		if a.Kind == "asset" {
			assets = append(assets, r)
			assetTotal += r.Balance
		} else {
			liabilities = append(liabilities, r)
			liabilityTotal += r.Balance
		}
	}
	s.render(w, r, "accounts.html", map[string]any{
		"Title":          "帳戶總覽",
		"Crumbs":         []string{"帳戶"},
		"Assets":         assets,
		"Liabilities":    liabilities,
		"AssetTotal":     assetTotal,
		"LiabilityTotal": liabilityTotal,
		"NetWorth":       assetTotal - liabilityTotal,
		"Active":         "accounts",
	})
}

func (s *Server) accountCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.error500(w, err)
		return
	}
	name := r.FormValue("name")
	kind := r.FormValue("kind")
	if name == "" || (kind != "asset" && kind != "liability") {
		http.Error(w, "name and kind required", http.StatusBadRequest)
		return
	}
	_, err := models.CreateAccount(s.DB, &models.Account{
		Name:   name,
		Kind:   kind,
		Active: true,
	})
	if err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (s *Server) accountUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if err := r.ParseForm(); err != nil {
		s.error500(w, err)
		return
	}
	a, err := models.GetAccount(s.DB, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if v := r.FormValue("name"); v != "" {
		a.Name = v
	}
	if v := r.FormValue("kind"); v == "asset" || v == "liability" {
		a.Kind = v
	}
	a.Active = r.FormValue("active") == "on" || r.FormValue("active") == "1"
	if err := models.UpdateAccount(s.DB, a); err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}

func (s *Server) accountDelete(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if err := models.DeleteAccount(s.DB, id); err != nil {
		auth.WriteAlertRedirect(w, r, http.StatusConflict,
			"刪除失敗：此帳戶仍有交易記錄。", "/accounts")
		return
	}
	http.Redirect(w, r, "/accounts", http.StatusSeeOther)
}
