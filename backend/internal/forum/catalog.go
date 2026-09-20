package forum

import (
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type Section struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Archived    bool   `json:"is_archived"`
	TopicCount  int    `json:"topic_count"`
}
type Topic struct {
	ID           string    `json:"id"`
	SectionID    string    `json:"section_id"`
	SectionTitle string    `json:"section_title"`
	SectionSlug  string    `json:"section_slug"`
	Archived     bool      `json:"section_archived"`
	Title        string    `json:"title"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	LastActivity time.Time `json:"last_activity_at"`
	PostCount    int       `json:"post_count"`
	Preview      string    `json:"preview"`
}
type Post struct {
	ID        string    `json:"id"`
	Number    int       `json:"number"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	IsMine    bool      `json:"is_mine"`
}

func pagination(r *http.Request) (int, error) {
	v := r.URL.Query().Get("page")
	if v == "" {
		return 1, nil
	}
	n, e := strconv.Atoi(v)
	if e != nil || n < 1 || n > 100000 {
		return 0, problem(400, "validation", "Некорректный номер страницы.")
	}
	return n, nil
}
func validText(s string, min, max int) bool {
	n := utf8.RuneCountInString(s)
	return n >= min && n <= max
}
func (s *Server) sections(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `SELECT s.id,s.slug,s.title,s.description,s.is_archived,(SELECT count(*) FROM topics t WHERE t.section_id=s.id) FROM sections s ORDER BY CASE s.slug WHEN 'general' THEN 1 WHEN 'tech' THEN 2 WHEN 'study' THEN 3 WHEN 'life' THEN 4 WHEN 'ideas' THEN 5 ELSE 6 END,s.created_at,s.id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []Section{}
	for rows.Next() {
		var x Section
		if err = rows.Scan(&x.ID, &x.Slug, &x.Title, &x.Description, &x.Archived, &x.TopicCount); err != nil {
			return err
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items})
}

const topicSelect = `SELECT t.id,t.section_id,s.title,s.slug,s.is_archived,t.title,t.status,t.created_at,t.last_activity_at,
 (SELECT count(*) FROM posts p WHERE p.topic_id=t.id),
 COALESCE((SELECT left(p.body,200) FROM posts p WHERE p.topic_id=t.id AND p.status='visible' ORDER BY p.number LIMIT 1),'')
 FROM topics t JOIN sections s ON s.id=t.section_id `

func scanTopic(row pgx.Row) (Topic, error) {
	var t Topic
	err := row.Scan(&t.ID, &t.SectionID, &t.SectionTitle, &t.SectionSlug, &t.Archived, &t.Title, &t.Status, &t.CreatedAt, &t.LastActivity, &t.PostCount, &t.Preview)
	return t, err
}
func (s *Server) topics(w http.ResponseWriter, r *http.Request) error {
	page, err := pagination(r)
	if err != nil {
		return err
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	section := r.URL.Query().Get("section_id")
	if q != "" && !validText(q, 2, 100) {
		return problem(400, "validation", "Поисковый запрос — от 2 до 100 символов.")
	}
	if section != "" && !uuidPattern.MatchString(section) {
		return problem(400, "validation", "Некорректный раздел.")
	}
	order := "t.last_activity_at DESC,t.id DESC"
	if r.URL.Query().Get("sort") == "new" {
		order = "t.created_at DESC,t.id DESC"
	}
	filter := `WHERE ($1='' OR t.section_id::text=$1) AND ($2='' OR t.search_vector @@ plainto_tsquery('russian',$2) OR EXISTS (SELECT 1 FROM posts p WHERE p.topic_id=t.id AND p.status='visible' AND p.search_vector @@ plainto_tsquery('russian',$2)))`
	var total int
	if err = s.db.QueryRow(r.Context(), "SELECT count(*) FROM topics t "+filter, section, q).Scan(&total); err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), topicSelect+filter+" ORDER BY "+order+" LIMIT 20 OFFSET $3", section, q, (page-1)*20)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []Topic{}
	for rows.Next() {
		t, e := scanTopic(rows)
		if e != nil {
			return e
		}
		items = append(items, t)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items, "page": page, "page_size": 20, "total": total})
}
func (s *Server) posts(w http.ResponseWriter, r *http.Request) error {
	page, err := pagination(r)
	if err != nil {
		return err
	}
	t, err := scanTopic(s.db.QueryRow(r.Context(), topicSelect+" WHERE t.id=$1", r.PathValue("id")))
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `SELECT id,number,CASE WHEN status='visible' THEN body ELSE '' END,status,created_at,COALESCE(author_session_id::text=$2,false) FROM posts WHERE topic_id=$1 ORDER BY number LIMIT 20 OFFSET $3`, t.ID, state(r).Anon.ID, (page-1)*20)
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []Post{}
	for rows.Next() {
		var p Post
		if err = rows.Scan(&p.ID, &p.Number, &p.Body, &p.Status, &p.CreatedAt, &p.IsMine); err != nil {
			return err
		}
		items = append(items, p)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"topic": t, "items": items, "page": page, "page_size": 20, "total": t.PostCount})
}
