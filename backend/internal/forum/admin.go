package forum

import (
	"net/http"
	"strings"
)

type sectionInput struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Archived    bool   `json:"is_archived"`
}

func validateSection(in *sectionInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Slug = strings.TrimSpace(in.Slug)
	in.Description = strings.TrimSpace(in.Description)
	if !slugPattern.MatchString(in.Slug) || !validText(in.Title, 2, 80) || !validText(in.Description, 0, 240) {
		return problem(400, "validation", "Название: 2–80 символов; адрес: 2–40 латинских букв, цифр или дефисов; описание: до 240.")
	}
	return nil
}

func (s *Server) createSection(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in sectionInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := validateSection(&in); err != nil {
		return err
	}
	id, err := s.insertSection(r.Context(), in)
	if err != nil {
		return err
	}
	return respond(w, 201, map[string]string{"id": id})
}

func (s *Server) updateSection(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in sectionInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := validateSection(&in); err != nil {
		return err
	}
	if err := s.updateSectionRecord(r.Context(), r.PathValue("id"), in); err != nil {
		return err
	}
	return respond(w, 204, nil)
}

func (s *Server) listStaff(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	items, err := s.queryStaff(r.Context())
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items})
}

func (s *Server) addStaff(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := validateStaffCredentials(in.Login, in.Password, in.Role); err != nil {
		return problem(400, "validation", "Логин: 3–40 латинских символов; пароль: 12–128 байт; роль: moderator/admin.")
	}
	if err := CreateStaff(r.Context(), s.db, in.Login, in.Password, in.Role); err != nil {
		return err
	}
	return respond(w, 201, map[string]string{"login": in.Login})
}

func (s *Server) changeRole(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in struct {
		Role string `json:"role"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if in.Role != "admin" && in.Role != "moderator" {
		return problem(400, "validation", "Неизвестная роль.")
	}
	if err := s.setStaffRole(r.Context(), r.PathValue("id"), in.Role); err != nil {
		return err
	}
	return respond(w, 204, nil)
}
