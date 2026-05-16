package handlers

import (
	"net/http"

	"github.com/hcchien/reviz-accounting/internal/models"
)

func (s *Server) projectsList(w http.ResponseWriter, r *http.Request) {
	projs, err := models.ListProjects(s.DB)
	if err != nil {
		s.error500(w, err)
		return
	}
	s.render(w, r, "projects.html", map[string]any{
		"Title":    "專案",
		"Crumbs":   []string{"專案"},
		"Projects": projs,
		"Active":   "projects",
	})
}

func (s *Server) projectCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.error500(w, err)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	_, err := models.CreateProject(s.DB, &models.Project{
		Name:      name,
		StartDate: models.NullStringFrom(r.FormValue("start_date")),
		EndDate:   models.NullStringFrom(r.FormValue("end_date")),
		Note:      r.FormValue("note"),
	})
	if err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) projectUpdate(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if err := r.ParseForm(); err != nil {
		s.error500(w, err)
		return
	}
	p, err := models.GetProject(s.DB, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if v := r.FormValue("name"); v != "" {
		p.Name = v
	}
	p.StartDate = models.NullStringFrom(r.FormValue("start_date"))
	p.EndDate = models.NullStringFrom(r.FormValue("end_date"))
	p.Note = r.FormValue("note")
	if err := models.UpdateProject(s.DB, p); err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (s *Server) projectDelete(w http.ResponseWriter, r *http.Request) {
	id := parseInt64(r.PathValue("id"))
	if err := models.DeleteProject(s.DB, id); err != nil {
		s.error500(w, err)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}
