package forum

import (
	"net/http"
	"strings"
)

type reportDecisionInput struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}

func (in *reportDecisionInput) validate() error {
	in.Reason = strings.TrimSpace(in.Reason)
	if (in.Decision != "hide" && in.Decision != "dismiss") || !validText(in.Reason, 1, 1000) {
		return problem(400, "validation", "Укажите решение и причину (1–1000 символов).")
	}
	return nil
}

type topicStatusInput struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func (in *topicStatusInput) validate() error {
	in.Reason = strings.TrimSpace(in.Reason)
	if (in.Status != "open" && in.Status != "closed") || !validText(in.Reason, 1, 1000) {
		return problem(400, "validation", "Укажите статус и причину (1–1000 символов).")
	}
	return nil
}

func (s *Server) modReports(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "moderator"); err != nil {
		return err
	}
	page, err := pagination(r)
	if err != nil {
		return err
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "open"
	}
	if status != "open" && status != "dismissed" && status != "resolved" {
		return problem(400, "validation", "Неизвестный статус жалобы.")
	}
	items, total, err := s.queryModerationReports(r.Context(), status, page)
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items, "page": page, "total": total, "page_size": 20})
}

func (s *Server) decideReport(w http.ResponseWriter, r *http.Request) error {
	staff, err := s.require(r, "moderator")
	if err != nil {
		return err
	}
	var in reportDecisionInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	status, err := s.applyReportDecision(r.Context(), staff.ID, r.PathValue("id"), in)
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"status": status})
}

func (s *Server) changeTopic(w http.ResponseWriter, r *http.Request) error {
	staff, err := s.require(r, "moderator")
	if err != nil {
		return err
	}
	var in topicStatusInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := in.validate(); err != nil {
		return err
	}
	if err := s.setTopicStatus(r.Context(), staff.ID, r.PathValue("id"), in); err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"status": in.Status})
}

func (s *Server) modActions(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "moderator"); err != nil {
		return err
	}
	page, err := pagination(r)
	if err != nil {
		return err
	}
	items, total, err := s.queryModerationActions(r.Context(), page)
	if err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items, "total": total, "page": page, "page_size": 20})
}
