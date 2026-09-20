package forum

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"strings"
	"sync"
	"testing"
)

type testClient struct {
	t          *testing.T
	client     *http.Client
	base, csrf string
}

func (c *testClient) call(method, path string, body any, key string) (int, map[string]any, http.Header) {
	c.t.Helper()
	var b io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		b = bytes.NewReader(data)
	}
	req, _ := http.NewRequest(method, c.base+"/api/v1"+path, b)
	req.Header.Set("Origin", c.base)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", c.csrf)
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	if len(data) > 0 {
		if err = json.Unmarshal(data, &out); err != nil {
			c.t.Fatalf("%s %s returned invalid JSON: %s", method, path, data)
		}
	}
	return resp.StatusCode, out, resp.Header
}
func expect(t *testing.T, got, want int, out map[string]any) {
	t.Helper()
	if got != want {
		t.Fatalf("status %d, expected %d: %#v", got, want, out)
	}
}
func client(t *testing.T, base string) *testClient {
	j, _ := cookiejar.New(nil)
	return &testClient{t: t, base: base, client: &http.Client{Jar: j}}
}
func TestIntegrationForum(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL; make integration prepares isolated PostgreSQL database")
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil || !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		t.Fatal("integration database name must end in _test")
	}
	ctx := context.Background()
	db, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err = Migrate(ctx, db); err != nil {
		t.Fatal("migration is not idempotent", err)
	}
	if _, err = db.Exec(ctx, "TRUNCATE sections,anonymous_sessions,staff_accounts CASCADE"); err != nil {
		t.Fatal(err)
	}
	if err = Seed(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, r := range []string{"admin", "moderator"} {
		if err = CreateStaff(ctx, db, r, "long-test-password-123", r); err != nil {
			t.Fatal(err)
		}
	}
	app := New(db, Config{Secret: strings.Repeat("test-secret", 5)})
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	app.cfg.Origin = server.URL
	a, b, admin, mod := client(t, server.URL), client(t, server.URL), client(t, server.URL), client(t, server.URL)
	for _, c := range []*testClient{a, b} {
		code, out, _ := c.call("POST", "/sessions", nil, "")
		expect(t, code, 200, out)
		c.csrf = out["csrf_token"].(string)
	}
	for _, c := range []*testClient{admin, mod} {
		login := "admin"
		if c == mod {
			login = "moderator"
		}
		code, out, _ := c.call("POST", "/staff/session", map[string]string{"login": login, "password": "long-test-password-123"}, "")
		expect(t, code, 200, out)
		c.csrf = out["staff"].(map[string]any)["csrf_token"].(string)
	}
	code, out, _ := a.call("GET", "/sections", nil, "")
	expect(t, code, 200, out)
	section := out["items"].([]any)[0].(map[string]any)
	sid := section["id"].(string)
	clearRates := func() {
		t.Helper()
		if _, err := db.Exec(ctx, "DELETE FROM rate_events"); err != nil {
			t.Fatal(err)
		}
	}
	payload := map[string]string{"section_id": sid, "title": "Интеграционная проверка", "body": "Первое сообщение"}
	t.Run("validation_and_csrf", func(t *testing.T) {
		bad := map[string]string{"section_id": sid, "title": "1234", "body": "текст"}
		code, out, _ := a.call("POST", "/topics", bad, "invalid-title")
		expect(t, code, 400, out)
		old := a.csrf
		a.csrf = "bad"
		code, out, _ = a.call("POST", "/topics", payload, "invalid-csrf")
		expect(t, code, 403, out)
		a.csrf = old
		req, _ := http.NewRequest("DELETE", server.URL+"/api/v1/session", nil)
		req.Header.Set("Origin", "https://other.example")
		req.Header.Set("X-CSRF-Token", a.csrf)
		res, err := a.client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != 403 {
			t.Fatal("cross-site mutation accepted")
		}
	})
	code, out, _ = a.call("POST", "/topics", payload, "create-topic-key")
	expect(t, code, 201, out)
	tid := out["topic_id"].(string)
	pid := out["first_post_id"].(string)
	t.Run("idempotency_and_rate_limit", func(t *testing.T) {
		code, replay, h := a.call("POST", "/topics", payload, "create-topic-key")
		expect(t, code, 201, replay)
		if replay["topic_id"] != tid || h.Get("Idempotency-Replayed") != "true" {
			t.Fatal("duplicate topic created")
		}
		code, out, h := a.call("POST", "/topics", payload, "another-topic-key")
		expect(t, code, 429, out)
		if h.Get("Retry-After") == "" {
			t.Fatal("Retry-After missing")
		}
		changed := map[string]string{"section_id": sid, "title": "Другое название", "body": "изменено"}
		code, out, _ = a.call("POST", "/topics", changed, "create-topic-key")
		expect(t, code, 409, out)
	})
	t.Run("ownership_and_private_dto", func(t *testing.T) {
		code, out, _ := b.call("DELETE", "/posts/"+pid, nil, "")
		expect(t, code, 403, out)
		code, out, _ = b.call("GET", "/topics/"+tid+"/posts", nil, "")
		expect(t, code, 200, out)
		raw, _ := json.Marshal(out)
		for _, secret := range []string{"author_session_id", "token_hash", "reporter_session_id", "password_hash"} {
			if bytes.Contains(raw, []byte(secret)) {
				t.Fatal("private field leaked", secret)
			}
		}
		if out["items"].([]any)[0].(map[string]any)["is_mine"] != false {
			t.Fatal("ownership leaked between visitors")
		}
	})
	var replyID string
	t.Run("reply_and_parallel_idempotency", func(t *testing.T) {
		body := map[string]string{"body": "Скрываемыйуникум <script>alert(1)</script>"}
		code, out, _ := a.call("POST", "/topics/"+tid+"/posts", body, "reply-key-0001")
		expect(t, code, 201, out)
		replyID = out["post_id"].(string)
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				code, out, _ := a.call("POST", "/topics/"+tid+"/posts", body, "reply-key-0001")
				expect(t, code, 201, out)
				if out["post_id"] != replyID {
					t.Error("concurrent duplicate")
				}
			}()
		}
		wg.Wait()
		code, out, _ = a.call("POST", "/topics/"+tid+"/posts", body, "reply-key-0002")
		expect(t, code, 429, out)
	})
	var rid string
	t.Run("reports", func(t *testing.T) {
		body := map[string]string{"post_id": replyID, "reason": "abuse", "comment": "Нужна проверка"}
		code, out, _ := b.call("POST", "/reports", body, "")
		expect(t, code, 201, out)
		rid = out["report_id"].(string)
		code, out, _ = b.call("POST", "/reports", body, "")
		expect(t, code, 409, out)
		code, out, _ = b.call("POST", "/reports", map[string]string{"post_id": replyID, "reason": "invalid"}, "")
		expect(t, code, 400, out)
		code, out, _ = b.call("POST", "/mod/reports/"+rid+"/decision", map[string]string{"decision": "hide", "reason": "bad"}, "")
		expect(t, code, 403, out)
		code, out, _ = mod.call("POST", "/mod/reports/"+rid+"/decision", map[string]string{"decision": "hide", "reason": "Нарушение правил"}, "")
		expect(t, code, 200, out)
		code, out, _ = a.call("GET", "/topics/"+tid+"/posts", nil, "")
		expect(t, code, 200, out)
		last := out["items"].([]any)[1].(map[string]any)
		if last["body"] != "" || last["status"] != "hidden" {
			t.Fatal("hidden text exposed")
		}
		code, out, _ = a.call("GET", "/topics?q=Скрываемыйуникум", nil, "")
		expect(t, code, 200, out)
		if out["total"] != float64(0) {
			t.Fatal("hidden text indexed in public search")
		}
		code, out, _ = mod.call("GET", "/mod/actions", nil, "")
		expect(t, code, 200, out)
		if out["total"] != float64(1) {
			t.Fatal("audit not persisted")
		}
	})
	t.Run("topic_status", func(t *testing.T) {
		code, out, _ := mod.call("PATCH", "/mod/topics/"+tid, map[string]string{"status": "closed", "reason": "Проверка"}, "")
		expect(t, code, 200, out)
		code, out, _ = b.call("POST", "/topics/"+tid+"/posts", map[string]string{"body": "новый ответ"}, "closed-reply-key")
		expect(t, code, 409, out)
		code, out, _ = mod.call("PATCH", "/mod/topics/"+tid, map[string]string{"status": "open", "reason": "Проверка завершена"}, "")
		expect(t, code, 200, out)
	})
	t.Run("admin_roles_and_archive", func(t *testing.T) {
		body := map[string]any{"slug": "test-section", "title": "Тестовый раздел", "description": "проверка", "is_archived": false}
		code, out, _ := mod.call("POST", "/admin/sections", body, "")
		expect(t, code, 403, out)
		code, out, _ = admin.call("POST", "/admin/sections", body, "")
		expect(t, code, 201, out)
		id := out["id"].(string)
		body["is_archived"] = true
		code, out, _ = admin.call("PATCH", "/admin/sections/"+id, body, "")
		expect(t, code, 204, out)
		clearRates()
		code, out, _ = a.call("POST", "/topics", map[string]string{"section_id": id, "title": "Архивная тема", "body": "текст"}, "archived-topic-key")
		expect(t, code, 409, out)
		code, out, _ = admin.call("GET", "/admin/staff", nil, "")
		expect(t, code, 200, out)
		var aid, mid string
		for _, row := range out["items"].([]any) {
			x := row.(map[string]any)
			if x["role"] == "admin" {
				aid = x["id"].(string)
			} else {
				mid = x["id"].(string)
			}
		}
		code, out, _ = admin.call("PATCH", "/admin/staff/"+aid+"/role", map[string]string{"role": "moderator"}, "")
		expect(t, code, 409, out)
		code, out, _ = admin.call("PATCH", "/admin/staff/"+mid+"/role", map[string]string{"role": "admin"}, "")
		expect(t, code, 204, out)
		code, out, _ = mod.call("GET", "/mod/reports", nil, "")
		expect(t, code, 403, out)
	})
	t.Run("delete_and_session_revocation", func(t *testing.T) {
		code, out, _ := a.call("DELETE", "/posts/"+pid, nil, "")
		expect(t, code, 204, out)
		code, out, _ = a.call("GET", "/topics/"+tid+"/posts", nil, "")
		expect(t, code, 200, out)
		post := out["items"].([]any)[0].(map[string]any)
		if post["body"] != "" || post["status"] != "deleted" {
			t.Fatal("body not erased")
		}
		cookieURL, _ := neturl.Parse(server.URL)
		oldCookies := a.client.Jar.Cookies(cookieURL)
		code, out, _ = a.call("DELETE", "/session", nil, "")
		expect(t, code, 204, out)
		a.client.Jar.SetCookies(cookieURL, oldCookies)
		code, out, _ = a.call("DELETE", "/posts/"+pid, nil, "")
		expect(t, code, 403, out)
	})
	t.Run("transaction_rollback", func(t *testing.T) {
		clearRates()
		_, err := db.Exec(ctx, `CREATE FUNCTION test_reject_post() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$; CREATE TRIGGER test_post_failure BEFORE INSERT ON posts FOR EACH ROW EXECUTE FUNCTION test_reject_post()`)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Exec(ctx, "DROP TRIGGER test_post_failure ON posts; DROP FUNCTION test_reject_post()")
		code, out, _ := b.call("POST", "/topics", map[string]string{"section_id": sid, "title": "Транзакция должна откатиться", "body": "текст"}, "rollback-topic-key")
		expect(t, code, 500, out)
		var count int
		if err = db.QueryRow(ctx, "SELECT count(*) FROM topics WHERE title='Транзакция должна откатиться'").Scan(&count); err != nil || count != 0 {
			t.Fatal("orphan topic after rollback", err, count)
		}
	})
	t.Run("pagination_and_search", func(t *testing.T) {
		var section string
		if err = db.QueryRow(ctx, "INSERT INTO sections(slug,title) VALUES('pagination','Пагинация') RETURNING id").Scan(&section); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 25; i++ {
			if _, err = db.Exec(ctx, "INSERT INTO topics(section_id,title) VALUES($1,$2)", section, fmt.Sprintf("Тест пагинации %02d", i)); err != nil {
				t.Fatal(err)
			}
		}
		seen := map[string]bool{}
		for page := 1; page <= 2; page++ {
			code, out, _ := b.call("GET", fmt.Sprintf("/topics?section_id=%s&page=%d", section, page), nil, "")
			expect(t, code, 200, out)
			rows := out["items"].([]any)
			want := 20
			if page == 2 {
				want = 5
			}
			if len(rows) != want {
				t.Fatal("wrong page size")
			}
			for _, row := range rows {
				id := row.(map[string]any)["id"].(string)
				if seen[id] {
					t.Fatal("duplicate across pages")
				}
				seen[id] = true
			}
		}
		code, out, _ := b.call("GET", "/topics?q=%27%3BDROP%20TABLE%20posts%3B--", nil, "")
		expect(t, code, 200, out)
	})
}
