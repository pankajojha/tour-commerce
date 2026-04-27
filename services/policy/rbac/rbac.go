package rbac

import (
	"context"
	"database/sql"
	"fmt"
)

// HasPermission returns true if the given user has the named operation permission
// within the given company context (companyID may be empty for platform-level checks).
func HasPermission(ctx context.Context, db *sql.DB, userID, operation, companyID string) (bool, error) {
	var allowed bool
	var err error
	if companyID == "" {
		err = db.QueryRowContext(ctx,
			`SELECT user_has_permission($1, $2)`,
			userID, operation,
		).Scan(&allowed)
	} else {
		err = db.QueryRowContext(ctx,
			`SELECT user_has_permission($1, $2, $3::uuid)`,
			userID, operation, companyID,
		).Scan(&allowed)
	}
	if err != nil {
		return false, fmt.Errorf("rbac.HasPermission: %w", err)
	}
	return allowed, nil
}

// RoleLevel returns the effective role level for the user (1 = highest, 5 = lowest, 99 = none).
func RoleLevel(ctx context.Context, db *sql.DB, userID, companyID string) (int, error) {
	var level int
	var err error
	if companyID == "" {
		err = db.QueryRowContext(ctx,
			`SELECT user_role_level($1)`, userID,
		).Scan(&level)
	} else {
		err = db.QueryRowContext(ctx,
			`SELECT user_role_level($1, $2::uuid)`, userID, companyID,
		).Scan(&level)
	}
	if err != nil {
		return 99, fmt.Errorf("rbac.RoleLevel: %w", err)
	}
	return level, nil
}
