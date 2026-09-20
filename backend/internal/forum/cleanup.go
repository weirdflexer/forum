package forum

import (
	"context"
)

func (s *Server) Cleanup(ctx context.Context) {
	// FK ON DELETE SET NULL removes internal authorship once a session expires.
	for _, q := range []string{"DELETE FROM rate_events WHERE created_at < now()-interval '1 hour'", "DELETE FROM idempotency_keys WHERE expires_at<now()", "DELETE FROM staff_sessions WHERE expires_at<now()", "DELETE FROM anonymous_sessions WHERE expires_at<now() OR revoked_at IS NOT NULL"} {
		if _, err := s.db.Exec(ctx, q); err != nil {
			return
		}
	}
}
