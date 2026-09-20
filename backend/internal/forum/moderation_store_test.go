package forum

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestIntegrationModerationTransactions(t *testing.T) {
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
	if err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	s := &Server{db: db}

	t.Run("competing_decisions_commit_one_audit_entry", func(t *testing.T) {
		f := newModerationFixture(t, db)
		type result struct {
			status string
			err    error
		}
		start := make(chan struct{})
		results := make(chan result, 2)
		for _, decision := range []string{"hide", "dismiss"} {
			go func(decision string) {
				<-start
				status, err := s.applyReportDecision(ctx, f.staffID, f.reportID, reportDecisionInput{Decision: decision, Reason: decision + " reason"})
				results <- result{status, err}
			}(decision)
		}
		close(start)
		var winner string
		var accepted, rejected int
		for range 2 {
			r := <-results
			if r.err == nil {
				accepted++
				winner = r.status
				continue
			}
			var p *apiError
			if !errors.As(r.err, &p) || p.Status != 409 || p.Code != "already_resolved" {
				t.Errorf("unexpected decision error: %v", r.err)
			}
			rejected++
		}
		if accepted != 1 || rejected != 1 {
			t.Fatalf("accepted=%d rejected=%d; expected one of each", accepted, rejected)
		}
		var status, reason, postStatus, action string
		var count int
		if err := db.QueryRow(ctx, "SELECT r.status,r.decision_reason,p.status FROM reports r JOIN posts p ON p.id=r.post_id WHERE r.id=$1", f.reportID).Scan(&status, &reason, &postStatus); err != nil {
			t.Fatal(err)
		}
		if err := db.QueryRow(ctx, "SELECT count(*),min(action) FROM moderation_actions WHERE post_id=$1", f.postID).Scan(&count, &action); err != nil {
			t.Fatal(err)
		}
		expectedAction, expectedPostStatus := "dismiss", "visible"
		if winner == "resolved" {
			expectedAction, expectedPostStatus = "hide", "hidden"
		} else if winner != "dismissed" {
			t.Fatalf("unexpected final status: %q", winner)
		}
		if status != winner || reason != expectedAction+" reason" || postStatus != expectedPostStatus || action != expectedAction || count != 1 {
			t.Fatalf("inconsistent decision: status=%s reason=%q post=%s action=%s audit_count=%d", status, reason, postStatus, action, count)
		}
	})

	t.Run("audit_failure_rolls_back_report_and_topic", func(t *testing.T) {
		f := newModerationFixture(t, db)
		var missingStaffID string
		if err := db.QueryRow(ctx, "SELECT gen_random_uuid()::text").Scan(&missingStaffID); err != nil {
			t.Fatal(err)
		}
		_, err := s.applyReportDecision(ctx, missingStaffID, f.reportID, reportDecisionInput{Decision: "hide", Reason: "Нарушение правил"})
		assertModerationForeignKeyError(t, err)
		var reportStatus, postStatus, reason string
		var unresolved bool
		if err := db.QueryRow(ctx, "SELECT r.status,r.decision_reason,r.resolved_at IS NULL,p.status FROM reports r JOIN posts p ON p.id=r.post_id WHERE r.id=$1", f.reportID).Scan(&reportStatus, &reason, &unresolved, &postStatus); err != nil {
			t.Fatal(err)
		}
		if reportStatus != "open" || postStatus != "visible" || reason != "" || !unresolved {
			t.Fatalf("partial report transaction persisted: report=%s post=%s reason=%q unresolved=%v", reportStatus, postStatus, reason, unresolved)
		}
		err = s.setTopicStatus(ctx, missingStaffID, f.topicID, topicStatusInput{Status: "closed", Reason: "Нарушение правил"})
		assertModerationForeignKeyError(t, err)
		var topicStatus string
		if err := db.QueryRow(ctx, "SELECT status FROM topics WHERE id=$1", f.topicID).Scan(&topicStatus); err != nil {
			t.Fatal(err)
		}
		var auditCount int
		if err := db.QueryRow(ctx, "SELECT count(*) FROM moderation_actions WHERE post_id=$1 OR topic_id=$2", f.postID, f.topicID).Scan(&auditCount); err != nil {
			t.Fatal(err)
		}
		if topicStatus != "open" || auditCount != 0 {
			t.Fatalf("partial topic transaction persisted: status=%s audit_count=%d", topicStatus, auditCount)
		}
	})
}

type moderationFixture struct {
	staffID, topicID, postID, reportID string
}

func newModerationFixture(t *testing.T, db *pgxpool.Pool) moderationFixture {
	t.Helper()
	ctx := context.Background()
	// Every cleanup targets a generated ID, so fixtures from other tests survive.
	insert := func(query, cleanup string, args ...any) string {
		t.Helper()
		var id string
		if err := db.QueryRow(ctx, query, args...).Scan(&id); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if _, err := db.Exec(ctx, cleanup, id); err != nil {
				t.Errorf("remove moderation fixture: %v", err)
			}
		})
		return id
	}
	suffix := digest(token())[:16]
	f := moderationFixture{}
	f.staffID = insert("INSERT INTO staff_accounts(login,password_hash,role) VALUES($1,'unused','moderator') RETURNING id", "DELETE FROM staff_accounts WHERE id=$1", "mod-"+suffix)
	sectionID := insert("INSERT INTO sections(slug,title) VALUES($1,'Раздел') RETURNING id", "DELETE FROM sections WHERE id=$1", "mod-"+suffix)
	f.topicID = insert("INSERT INTO topics(section_id,title) VALUES($1,'Тест модерации') RETURNING id", "DELETE FROM topics WHERE id=$1", sectionID)
	f.postID = insert("INSERT INTO posts(topic_id,number,body) VALUES($1,1,'Сообщение') RETURNING id", "DELETE FROM posts WHERE id=$1", f.topicID)
	f.reportID = insert("INSERT INTO reports(post_id,reason) VALUES($1,'spam') RETURNING id", "DELETE FROM reports WHERE id=$1", f.postID)
	t.Cleanup(func() {
		if _, err := db.Exec(ctx, "DELETE FROM moderation_actions WHERE staff_id=$1", f.staffID); err != nil {
			t.Errorf("remove moderation audit fixture: %v", err)
		}
	})
	return f
}

func assertModerationForeignKeyError(t *testing.T, err error) {
	t.Helper()
	var p *pgconn.PgError
	if !errors.As(err, &p) || p.Code != "23503" {
		t.Fatalf("expected audit foreign key failure, got %v", err)
	}
}
