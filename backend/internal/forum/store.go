package forum

import (
	"context"
	"fmt"
	"forum/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
	"io/fs"
	"sort"
	"time"
)

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("invalid DATABASE_URL")
	}
	cfg.MaxConns = 12
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("PostgreSQL unavailable: %w", err)
	}
	return db, nil
}
func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(72830401)"); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); err != nil {
		return err
	}
	files, _ := fs.Glob(migrations.Files, "*.sql")
	sort.Strings(files)
	for _, name := range files {
		var done bool
		if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=$1)", name).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		b, _ := migrations.Files.ReadFile(name)
		if _, err = tx.Exec(ctx, string(b)); err != nil {
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES($1)", name); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func CreateStaff(ctx context.Context, db *pgxpool.Pool, login, password, role string) error {
	if !loginPattern.MatchString(login) {
		return fmt.Errorf("login: 3–40 latin letters, digits, underscores or hyphens")
	}
	if len(password) < 12 || len(password) > 128 {
		return fmt.Errorf("password must contain 12–128 bytes")
	}
	if role != "admin" && role != "moderator" {
		return fmt.Errorf("role must be admin or moderator")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO staff_accounts(login,password_hash,role) VALUES($1,$2,$3)", login, hash, role)
	return err
}
func Seed(ctx context.Context, db *pgxpool.Pool) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(72830402)")
	if err != nil {
		return err
	}
	var count int
	if err = tx.QueryRow(ctx, "SELECT count(*) FROM sections").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	sections := [][3]string{{"general", "Обо всём", "Мысли, вопросы и разговоры без повода"}, {"tech", "Технологии", "Код, инструменты и цифровой мир"}, {"study", "Учёба", "Помогаем разобраться и делимся опытом"}, {"life", "Жизнь", "Всё, что происходит за пределами экрана"}, {"ideas", "Идеи", "Место для того, что ещё только начинается"}}
	samples := [][3]string{{"С чего начинается хороший разговор?", "Здесь можно быть собой. Без профиля, имени и необходимости производить впечатление.\n\nРасскажите, какая мысль занимала вас сегодня. Или просто почитайте — это тоже участие.", "Иногда самые интересные разговоры начинаются с простого вопроса. Как прошёл ваш день?"}, {"Какой небольшой проект помог вам вырасти?", "Большие планы часто так и остаются планами. А маленький проект на выходные может научить гораздо большему.\n\nЧто вы сделали своими руками и чему это вас научило?", "Сделал бота, который напоминает поливать растения. Теперь растения живы, а я разобрался с API."}, {"Как вы справляетесь с дедлайнами?", "Когда несколько работ нужно сдать одновременно, сложно понять, с чего начать.\n\nДелитесь работающими способами: списки, таймеры, совместная работа?", "Мне помогает разбить задачу до действия, которое можно сделать за 20 минут."}, {"Маленькие вещи, которые делают день лучше", "Тёплый чай, прогулка без телефона, любимый альбом. Иногда для хорошего дня нужно совсем немного.\n\nКакая маленькая привычка помогает вам?", "Выйти утром на десять минут раньше и пройтись пешком."}, {"Что бы вы создали, если бы не боялись ошибиться?", "Представим, что первая попытка не обязана быть идеальной.\n\nКакая идея давно ждёт своего часа?", "Место для обмена книгами в нашем районе. Наверное, начну с одной полки."}}
	for i, s := range sections {
		var sid, tid string
		if err = tx.QueryRow(ctx, "INSERT INTO sections(slug,title,description) VALUES($1,$2,$3) RETURNING id", s[0], s[1], s[2]).Scan(&sid); err != nil {
			return err
		}
		created := time.Now().Add(-time.Duration(i+1) * 3 * time.Hour)
		if err = tx.QueryRow(ctx, "INSERT INTO topics(section_id,title,created_at,last_activity_at) VALUES($1,$2,$3,$3) RETURNING id", sid, samples[i][0], created).Scan(&tid); err != nil {
			return err
		}
		for j := 1; j <= 2; j++ {
			if _, err = tx.Exec(ctx, "INSERT INTO posts(topic_id,number,body,created_at) VALUES($1,$2,$3,$4)", tid, j, samples[i][j], created.Add(time.Duration(j)*time.Minute)); err != nil {
				return err
			}
		}
	}
	if _, err = tx.Exec(ctx, "UPDATE topics t SET last_activity_at=(SELECT max(created_at) FROM posts p WHERE p.topic_id=t.id)"); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Server) Cleanup(ctx context.Context) {
	// FK ON DELETE SET NULL removes internal authorship once a session expires.
	for _, q := range []string{"DELETE FROM rate_events WHERE created_at < now()-interval '1 hour'", "DELETE FROM idempotency_keys WHERE expires_at<now()", "DELETE FROM staff_sessions WHERE expires_at<now()", "DELETE FROM anonymous_sessions WHERE expires_at<now() OR revoked_at IS NOT NULL"} {
		if _, err := s.db.Exec(ctx, q); err != nil {
			return
		}
	}
}
