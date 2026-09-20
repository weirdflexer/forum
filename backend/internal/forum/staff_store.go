package forum

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func CreateStaff(ctx context.Context, db *pgxpool.Pool, login, password, role string) error {
	if err := validateStaffCredentials(login, password, role); err != nil {
		return err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO staff_accounts(login,password_hash,role) VALUES($1,$2,$3)", login, hash, role)
	return err
}

func validateStaffCredentials(login, password, role string) error {
	if !loginPattern.MatchString(login) {
		return fmt.Errorf("login: 3–40 latin letters, digits, underscores or hyphens")
	}
	if len(password) < 12 || len(password) > 128 {
		return fmt.Errorf("password must contain 12–128 bytes")
	}
	if role != "admin" && role != "moderator" {
		return fmt.Errorf("role must be admin or moderator")
	}
	return nil
}
