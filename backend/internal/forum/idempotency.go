package forum

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

type publicationRequest struct {
	SessionID string
	Key       string
	Path      string
}

type publicationResult[T any] struct {
	Value  T
	Replay *savedPublication
}

type savedPublication struct {
	Status int
	Body   json.RawMessage
}

// publishOnce keeps the mutation, rate event and saved reply in one transaction.
// Replays are read after locking the session and before checking mutable target
// state or limits, so retrying an accepted write never performs it again.
func publishOnce[T any](ctx context.Context, s *Server, req publicationRequest, hash string, create func(pgx.Tx) (T, error)) (publicationResult[T], error) {
	var out publicationResult[T]
	tx, err := s.beginAnonymousWrite(ctx, req.SessionID)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out.Replay, err = loadPublication(ctx, tx, req.SessionID, req.Key, hash)
	if err != nil || out.Replay != nil {
		return out, err
	}
	out.Value, err = create(tx)
	if err != nil {
		return out, err
	}
	if err = savePublication(ctx, tx, req.SessionID, req.Key, hash, out.Value); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}

func loadPublication(ctx context.Context, tx pgx.Tx, sessionID, key, hash string) (*savedPublication, error) {
	var oldHash string
	var out savedPublication
	err := tx.QueryRow(ctx, "SELECT request_hash,response,status FROM idempotency_keys WHERE session_id=$1 AND key=$2 AND expires_at>now()", sessionID, key).Scan(&oldHash, &out.Body, &out.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if hash != oldHash {
		return nil, problem(409, "idempotency_conflict", "Этот ключ уже использован для другого запроса.")
	}
	return &out, nil
}

func savePublication(ctx context.Context, tx pgx.Tx, sessionID, key, hash string, result any) error {
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO idempotency_keys(session_id,key,request_hash,response,status,expires_at) VALUES($1,$2,$3,$4,201,now()+interval '24 hours') ON CONFLICT(session_id,key) DO UPDATE SET request_hash=excluded.request_hash,response=excluded.response,status=201,expires_at=excluded.expires_at`, sessionID, key, hash, body)
	return err
}

func requestHash(path string, value any) string {
	body, _ := json.Marshal(value)
	return digest(path + string(body))
}
