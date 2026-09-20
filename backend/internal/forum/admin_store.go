package forum

import "context"

func (s *Server) insertSection(ctx context.Context, in sectionInput) (string, error) {
	var id string
	err := s.db.QueryRow(ctx, "INSERT INTO sections(slug,title,description,is_archived) VALUES($1,$2,$3,$4) RETURNING id", in.Slug, in.Title, in.Description, in.Archived).Scan(&id)
	return id, err
}

func (s *Server) updateSectionRecord(ctx context.Context, id string, in sectionInput) error {
	tag, err := s.db.Exec(ctx, "UPDATE sections SET slug=$2,title=$3,description=$4,is_archived=$5 WHERE id=$1", id, in.Slug, in.Title, in.Description, in.Archived)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return problem(404, "not_found", "Раздел не найден.")
	}
	return nil
}

type staffAccount struct {
	ID    string `json:"id"`
	Login string `json:"login"`
	Role  string `json:"role"`
}

func (s *Server) queryStaff(ctx context.Context) ([]staffAccount, error) {
	rows, err := s.db.Query(ctx, "SELECT id,login,role FROM staff_accounts WHERE is_active ORDER BY login")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []staffAccount{}
	for rows.Next() {
		var item staffAccount
		if err := rows.Scan(&item.ID, &item.Login, &item.Role); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) setStaffRole(ctx context.Context, staffID, role string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// All role changes share this lock so concurrent demotions cannot remove
	// the last active administrator by both observing the same initial count.
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(72830403)"); err != nil {
		return err
	}
	var old string
	if err := tx.QueryRow(ctx, "SELECT role FROM staff_accounts WHERE id=$1 FOR UPDATE", staffID).Scan(&old); err != nil {
		return err
	}
	if old == "admin" && role != "admin" {
		var count int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM staff_accounts WHERE role='admin' AND is_active").Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return problem(409, "last_admin", "Нельзя убрать последнего администратора.")
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE staff_accounts SET role=$2 WHERE id=$1", staffID, role); err != nil {
		return err
	}
	// Existing logins must be invalidated atomically with the role change.
	if _, err := tx.Exec(ctx, "DELETE FROM staff_sessions WHERE staff_id=$1", staffID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
