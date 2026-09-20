package forum

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type sessionRateLimitError struct {
	RetryAfter int
	cause      error
}

func (e *sessionRateLimitError) Error() string { return e.cause.Error() }
func (e *sessionRateLimitError) Unwrap() error { return e.cause }

// Call only while holding the anonymous session row lock. The rate event rolls
// back with the surrounding write if validation or publication fails.
func sessionLimit(ctx context.Context, tx pgx.Tx, sessionID, action string, max, seconds int) error {
	var count int
	var oldest *time.Time
	err := tx.QueryRow(ctx, "SELECT count(*),min(created_at) FROM rate_events WHERE session_id=$1 AND action=$2 AND created_at>now()-make_interval(secs=>$3)", sessionID, action, seconds).Scan(&count, &oldest)
	if err != nil {
		return err
	}
	if count >= max {
		wait := seconds
		if oldest != nil {
			wait = int(time.Until(oldest.Add(time.Duration(seconds)*time.Second)).Seconds()) + 1
		}
		if wait < 1 {
			wait = 1
		}
		return &sessionRateLimitError{
			RetryAfter: wait,
			cause:      problem(429, "rate_limit", fmt.Sprintf("Подождите %d сек. перед следующим действием.", wait)),
		}
	}
	_, err = tx.Exec(ctx, "INSERT INTO rate_events(session_id,action) VALUES($1,$2)", sessionID, action)
	return err
}
