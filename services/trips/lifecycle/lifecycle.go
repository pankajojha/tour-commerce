package lifecycle

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TripStatus represents a valid lifecycle state.
type TripStatus string

const (
	StatusDraft     TripStatus = "draft"
	StatusActive    TripStatus = "active"
	StatusShared    TripStatus = "shared"
	StatusFinalized TripStatus = "finalized"
	StatusConsumed  TripStatus = "consumed"
)

// OperationRule holds a single row from trip_operation_rules.
type OperationRule struct {
	Operation     string
	Allowed       bool
	AllowedScopes []string
	Reason        string
}

// UIHint holds a single row from trip_ui_hints.
type UIHint struct {
	TripStatus      string `json:"trip_status"`
	BannerVariant   string `json:"banner_variant"`
	BannerMessage   string `json:"banner_message"`
	ShowEditActions bool   `json:"show_edit_actions"`
	ShowNewTripCTA  bool   `json:"show_new_trip_cta"`
	CTALabel        string `json:"cta_label,omitempty"`
	CTAAction       string `json:"cta_action,omitempty"`
}

// CheckOperation returns the OperationRule for the given trip status and operation.
// Returns an implicitly-denied rule when no row is found.
func CheckOperation(ctx context.Context, db *sql.DB, tripStatus, operation string) (*OperationRule, error) {
	var rule OperationRule
	var scopesRaw []byte
	var reason sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT operation, allowed, allowed_scopes, reason
		FROM trip_operation_rules
		WHERE trip_status = $1
		  AND operation   = $2
		  AND active      = TRUE
	`, tripStatus, operation).Scan(&rule.Operation, &rule.Allowed, &scopesRaw, &reason)

	if err == sql.ErrNoRows {
		return &OperationRule{Operation: operation, Allowed: false, Reason: "no rule defined"}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("CheckOperation query: %w", err)
	}
	if err = json.Unmarshal(scopesRaw, &rule.AllowedScopes); err != nil {
		return nil, fmt.Errorf("unmarshal allowed_scopes: %w", err)
	}
	if reason.Valid {
		rule.Reason = reason.String
	}
	return &rule, nil
}

// IsTransitionAllowed checks whether from->to is a legal status transition for the given actor scope.
func IsTransitionAllowed(ctx context.Context, db *sql.DB, fromStatus, toStatus, actorScope string) (bool, error) {
	var scopesRaw []byte

	err := db.QueryRowContext(ctx, `
		SELECT allowed_scopes
		FROM trip_status_transitions
		WHERE from_status = $1
		  AND to_status   = $2
		  AND active      = TRUE
	`, fromStatus, toStatus).Scan(&scopesRaw)

	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("IsTransitionAllowed query: %w", err)
	}

	var scopes []string
	if err = json.Unmarshal(scopesRaw, &scopes); err != nil {
		return false, fmt.Errorf("unmarshal allowed_scopes: %w", err)
	}
	for _, s := range scopes {
		if s == actorScope {
			return true, nil
		}
	}
	return false, nil
}

// GetUIHint returns the UI directive row for a given trip status.
func GetUIHint(ctx context.Context, db *sql.DB, tripStatus string) (*UIHint, error) {
	var hint UIHint
	var ctaLabel, ctaAction sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT trip_status, banner_variant, banner_message,
		       show_edit_actions, show_new_trip_cta, cta_label, cta_action
		FROM trip_ui_hints
		WHERE trip_status = $1 AND active = TRUE
	`, tripStatus).Scan(
		&hint.TripStatus, &hint.BannerVariant, &hint.BannerMessage,
		&hint.ShowEditActions, &hint.ShowNewTripCTA, &ctaLabel, &ctaAction,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetUIHint query: %w", err)
	}
	if ctaLabel.Valid {
		hint.CTALabel = ctaLabel.String
	}
	if ctaAction.Valid {
		hint.CTAAction = ctaAction.String
	}
	return &hint, nil
}

// AppendStatusLog inserts an immutable row into trip_status_change_log.
func AppendStatusLog(ctx context.Context, db *sql.DB,
	tripPlanID, changedByUserID uuid.UUID,
	fromStatus, toStatus, actorScope, source, correlationID, reason string,
) error {
	var fromVal interface{}
	if fromStatus != "" {
		fromVal = fromStatus
	}
	var byVal interface{}
	if changedByUserID != uuid.Nil {
		byVal = changedByUserID
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO trip_status_change_log
		    (trip_plan_id, from_status, to_status, changed_by_user_id,
		     actor_scope, reason, source, correlation_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
	`,
		tripPlanID, fromVal, toStatus, byVal,
		actorScope, nullStr(reason), nullStr(source), nullStr(correlationID),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("AppendStatusLog: %w", err)
	}
	return nil
}

// ActiveTrip holds the minimal fields returned by CheckOwnerHasActiveTrip.
type ActiveTrip struct {
	ID            string
	Status        string
	Title         string
	TripStartDate string // ISO-8601 DATE or empty string
	TripEndDate   string // ISO-8601 DATE or empty string
}

// CheckOwnerHasActiveTrip returns the owner's current non-consumed trip (if any).
// Returns nil, nil when the owner has no blocking trip (i.e. safe to create a new one).
// A non-nil *ActiveTrip means creation must be blocked.
func CheckOwnerHasActiveTrip(ctx context.Context, db *sql.DB, ownerUserID uuid.UUID) (*ActiveTrip, error) {
	var t ActiveTrip
	var startDate, endDate sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT id::text, status, title,
		       COALESCE(trip_start_date::text, ''),
		       COALESCE(trip_end_date::text, '')
		FROM trip_plans
		WHERE owner_user_id = $1
		  AND status IN ('draft', 'active', 'shared', 'finalized')
		LIMIT 1
	`, ownerUserID).Scan(&t.ID, &t.Status, &t.Title, &startDate, &endDate)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("CheckOwnerHasActiveTrip: %w", err)
	}
	t.TripStartDate = startDate.String
	t.TripEndDate = endDate.String
	return &t, nil
}

// CreateTrip inserts a new trip_plans row in 'draft' status and returns its UUID.
// tripStartDate and tripEndDate are optional ISO-8601 date strings ("" = NULL).
// Callers MUST call CheckOwnerHasActiveTrip first; this function relies on the
// uq_one_active_trip_per_owner unique index as a final DB-level guard and will
// return an error on duplicate-owner violations.
func CreateTrip(ctx context.Context, db *sql.DB, ownerUserID uuid.UUID, title, tripStartDate, tripEndDate string) (uuid.UUID, error) {
	if title == "" {
		title = "New Trip"
	}
	var newID uuid.UUID
	err := db.QueryRowContext(ctx, `
		INSERT INTO trip_plans (owner_user_id, title, status, trip_start_date, trip_end_date)
		VALUES ($1, $2, 'draft', $3, $4)
		RETURNING id
	`, ownerUserID, title, nullStr(tripStartDate), nullStr(tripEndDate)).Scan(&newID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("CreateTrip: %w", err)
	}
	return newID, nil
}

// ValidateTripDates checks that trip_end_date is set on the given trip.
// Returns a non-nil error (with a user-visible message) if the date is missing.
// Should be called before any transition out of 'draft'.
func ValidateTripDates(ctx context.Context, db *sql.DB, tripPlanID string) error {
	var endDate sql.NullString
	err := db.QueryRowContext(ctx,
		`SELECT trip_end_date::text FROM trip_plans WHERE id = $1`, tripPlanID,
	).Scan(&endDate)
	if err == sql.ErrNoRows {
		return fmt.Errorf("trip not found: %s", tripPlanID)
	}
	if err != nil {
		return fmt.Errorf("ValidateTripDates: %w", err)
	}
	if !endDate.Valid || endDate.String == "" {
		return fmt.Errorf("trip_end_date is required before the trip can leave draft status")
	}
	return nil
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
