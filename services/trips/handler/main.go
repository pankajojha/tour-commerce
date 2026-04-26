package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	tripdb "github.com/explorer-agent/trips/db"
	"github.com/explorer-agent/trips/lifecycle"
)

var db *sql.DB

func init() {
	var err error
	db, err = tripdb.Open()
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
}

// ─────────────────────────────────────────────
// Request / response shapes
// ─────────────────────────────────────────────

// TransitionRequest is the POST /trips/{id}/transition payload.
type TransitionRequest struct {
	ToStatus      string `json:"to_status"`
	ActorScope    string `json:"actor_scope"`
	Reason        string `json:"reason,omitempty"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// TransitionResponse is returned by the transition endpoint.
type TransitionResponse struct {
	TripPlanID string `json:"trip_plan_id"`
	FromStatus string `json:"from_status"`
	ToStatus   string `json:"to_status"`
	Allowed    bool   `json:"allowed"`
	Message    string `json:"message,omitempty"`
}

// OperationsResponse wraps all per-operation rules for the trip's current status.
type OperationsResponse struct {
	TripStatus string                     `json:"trip_status"`
	Operations map[string]OperationDetail `json:"operations"`
}

// CreateTripRequest is the POST /trips payload.
type CreateTripRequest struct {
	Title         string `json:"title"`
	TripStartDate string `json:"trip_start_date,omitempty"` // ISO-8601 DATE, e.g. "2025-09-01"
	TripEndDate   string `json:"trip_end_date,omitempty"`   // ISO-8601 DATE, e.g. "2025-09-14"
}

// CreateTripResponse is returned by POST /trips.
type CreateTripResponse struct {
	TripPlanID    string `json:"trip_plan_id"`
	Status        string `json:"status"`
	Title         string `json:"title"`
	TripStartDate string `json:"trip_start_date,omitempty"`
	TripEndDate   string `json:"trip_end_date,omitempty"`
}

// ActiveTripResponse is returned by GET /trips/owner/active.
type ActiveTripResponse struct {
	HasActiveTrip bool   `json:"has_active_trip"`
	TripPlanID    string `json:"trip_plan_id,omitempty"`
	Status        string `json:"status,omitempty"`
	Title         string `json:"title,omitempty"`
	TripStartDate string `json:"trip_start_date,omitempty"`
	TripEndDate   string `json:"trip_end_date,omitempty"`
}

// PatchTripRequest is the PATCH /trips/{id} payload.
// Only title, trip_start_date, and trip_end_date may be updated.
// Only valid while the trip is in 'draft' status.
type PatchTripRequest struct {
	Title         *string `json:"title,omitempty"`
	TripStartDate *string `json:"trip_start_date,omitempty"`
	TripEndDate   *string `json:"trip_end_date,omitempty"`
}

// OperationDetail is a single operation permission row.
type OperationDetail struct {
	Allowed       bool     `json:"allowed"`
	AllowedScopes []string `json:"allowed_scopes"`
	Reason        string   `json:"reason,omitempty"`
}

// ─────────────────────────────────────────────
// Router
// ─────────────────────────────────────────────

type ServiceHandler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	routes := map[string]ServiceHandler{
		"POST /trips":                  handleCreateTrip,
		"GET /trips/owner/active":      handleGetOwnerActiveTrip,
		"GET /trips/{id}/operations":   handleGetOperations,
		"POST /trips/{id}/transition":  handleTransition,
		"PATCH /trips/{id}":            handlePatchTrip,
		"GET /trips/ui-hints/{status}": handleGetUIHint,
	}

	key := matchRoute(req.HTTPMethod, req.Path)
	if h, ok := routes[key]; ok {
		return h(ctx, req)
	}
	return jsonError(http.StatusNotFound, "route not found"), nil
}

func matchRoute(method, path string) string {
	m := strings.ToUpper(method)
	switch {
	case strings.HasPrefix(path, "/trips/ui-hints/"):
		return m + " /trips/ui-hints/{status}"
	case path == "/trips/owner/active":
		return m + " /trips/owner/active"
	case path == "/trips":
		return m + " /trips"
	case strings.HasSuffix(path, "/operations"):
		return m + " /trips/{id}/operations"
	case strings.HasSuffix(path, "/transition"):
		return m + " /trips/{id}/transition"
	case m == "PATCH" && strings.Count(path, "/") == 2:
		return "PATCH /trips/{id}"
	}
	return m + " " + path
}

// ─────────────────────────────────────────────
// GET /trips/{id}/operations?scope=owner
// ─────────────────────────────────────────────

func handleGetOperations(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tripPlanID := req.PathParameters["id"]
	if _, err := uuid.Parse(tripPlanID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid trip id"), nil
	}

	actorScope := req.QueryStringParameters["scope"]
	if actorScope == "" {
		actorScope = "owner"
	}

	tripStatus, err := getTripStatus(ctx, tripPlanID)
	if err != nil {
		log.Printf("getTripStatus(%s): %v", tripPlanID, err)
		return jsonError(http.StatusInternalServerError, "failed to load trip"), nil
	}
	if tripStatus == "" {
		return jsonError(http.StatusNotFound, "trip not found"), nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT operation, allowed, allowed_scopes, COALESCE(reason,'')
		FROM trip_operation_rules
		WHERE trip_status = $1 AND active = TRUE
		ORDER BY operation
	`, tripStatus)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "db error"), nil
	}
	defer rows.Close()

	ops := make(map[string]OperationDetail)
	for rows.Next() {
		var op string
		var allowed bool
		var scopesRaw []byte
		var reason string
		if err = rows.Scan(&op, &allowed, &scopesRaw, &reason); err != nil {
			continue
		}
		var scopes []string
		_ = json.Unmarshal(scopesRaw, &scopes)
		ops[op] = OperationDetail{
			Allowed:       allowed && scopeIn(actorScope, scopes),
			AllowedScopes: scopes,
			Reason:        reason,
		}
	}

	return jsonOK(OperationsResponse{TripStatus: tripStatus, Operations: ops})
}

// ─────────────────────────────────────────────
// POST /trips/{id}/transition
// ─────────────────────────────────────────────

func handleTransition(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tripPlanID := req.PathParameters["id"]
	if _, err := uuid.Parse(tripPlanID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid trip id"), nil
	}

	var body TransitionRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.ToStatus == "" || body.ActorScope == "" {
		return jsonError(http.StatusBadRequest, "to_status and actor_scope are required"), nil
	}

	currentStatus, err := getTripStatus(ctx, tripPlanID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to load trip"), nil
	}
	if currentStatus == "" {
		return jsonError(http.StatusNotFound, "trip not found"), nil
	}

	allowed, err := lifecycle.IsTransitionAllowed(ctx, db, currentStatus, body.ToStatus, body.ActorScope)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "transition check failed"), nil
	}
	if !allowed {
		return jsonOK(TransitionResponse{
			TripPlanID: tripPlanID,
			FromStatus: currentStatus,
			ToStatus:   body.ToStatus,
			Allowed:    false,
			Message:    fmt.Sprintf("transition from %s to %s is not permitted for scope %s", currentStatus, body.ToStatus, body.ActorScope),
		})
	}

	// Guard: trip_end_date must be set before a trip can leave 'draft'.
	if currentStatus == "draft" {
		if dateErr := lifecycle.ValidateTripDates(ctx, db, tripPlanID); dateErr != nil {
			return jsonError(http.StatusUnprocessableEntity, dateErr.Error()), nil
		}
	}

	_, err = db.ExecContext(ctx,
		`UPDATE trip_plans SET status = $1, updated_at = NOW() WHERE id = $2`,
		body.ToStatus, tripPlanID,
	)
	if err != nil {
		log.Printf("update trip_plans status: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to update trip status"), nil
	}

	actorUserID := uuid.Nil
	if sub, ok := req.RequestContext.Authorizer["sub"].(string); ok {
		if parsed, pErr := uuid.Parse(sub); pErr == nil {
			actorUserID = parsed
		}
	}

	tripID, _ := uuid.Parse(tripPlanID)
	if logErr := lifecycle.AppendStatusLog(ctx, db,
		tripID, actorUserID,
		currentStatus, body.ToStatus,
		body.ActorScope, "service",
		body.CorrelationID, body.Reason,
	); logErr != nil {
		log.Printf("AppendStatusLog warning (non-fatal): %v", logErr)
	}

	return jsonOK(TransitionResponse{
		TripPlanID: tripPlanID,
		FromStatus: currentStatus,
		ToStatus:   body.ToStatus,
		Allowed:    true,
	})
}

// ─────────────────────────────────────────────
// POST /trips
// Creates a new trip for the authenticated owner.
// Returns HTTP 409 Conflict when the owner already has a non-consumed trip.
// ─────────────────────────────────────────────

func handleCreateTrip(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ownerUserID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body CreateTripRequest
	if req.Body != "" {
		if jsonErr := json.Unmarshal([]byte(req.Body), &body); jsonErr != nil {
			return jsonError(http.StatusBadRequest, "invalid request body"), nil
		}
	}

	// Enforce single-trip-per-owner: reject if a non-consumed trip already exists.
	existing, err := lifecycle.CheckOwnerHasActiveTrip(ctx, db, ownerUserID)
	if err != nil {
		log.Printf("CheckOwnerHasActiveTrip: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to check existing trips"), nil
	}
	if existing != nil {
		b, _ := json.Marshal(map[string]string{
			"error":        "a non-consumed trip already exists for this user",
			"trip_plan_id": existing.ID,
			"status":       existing.Status,
			"title":        existing.Title,
			"hint":         "complete or consume your current trip before starting a new one",
		})
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusConflict,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       string(b),
		}, nil
	}

	newID, err := lifecycle.CreateTrip(ctx, db, ownerUserID, body.Title, body.TripStartDate, body.TripEndDate)
	if err != nil {
		log.Printf("CreateTrip: %v", err)
		// Surface duplicate-owner DB-level violation as 409 too.
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return jsonError(http.StatusConflict, "a non-consumed trip already exists for this user"), nil
		}
		return jsonError(http.StatusInternalServerError, "failed to create trip"), nil
	}

	title := body.Title
	if title == "" {
		title = "New Trip"
	}
	return jsonOK(CreateTripResponse{
		TripPlanID:    newID.String(),
		Status:        "draft",
		Title:         title,
		TripStartDate: body.TripStartDate,
		TripEndDate:   body.TripEndDate,
	})
}

// ─────────────────────────────────────────────
// GET /trips/owner/active
// Returns the caller's current non-consumed trip summary (if any).
// ─────────────────────────────────────────────

func handleGetOwnerActiveTrip(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ownerUserID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	existing, err := lifecycle.CheckOwnerHasActiveTrip(ctx, db, ownerUserID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to check active trips"), nil
	}
	if existing == nil {
		return jsonOK(ActiveTripResponse{HasActiveTrip: false})
	}
	return jsonOK(ActiveTripResponse{
		HasActiveTrip: true,
		TripPlanID:    existing.ID,
		Status:        existing.Status,
		Title:         existing.Title,
		TripStartDate: existing.TripStartDate,
		TripEndDate:   existing.TripEndDate,
	})
}

// ─────────────────────────────────────────────
// GET /trips/ui-hints/{status}
// ─────────────────────────────────────────────

func handleGetUIHint(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	status := req.PathParameters["status"]
	if status == "" {
		parts := strings.Split(req.Path, "/")
		if len(parts) > 0 {
			status = parts[len(parts)-1]
		}
	}

	hint, err := lifecycle.GetUIHint(ctx, db, status)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "db error"), nil
	}
	if hint == nil {
		return jsonError(http.StatusNotFound, "no ui hint for status: "+status), nil
	}
	return jsonOK(hint)
}

// ─────────────────────────────────────────────
// PATCH /trips/{id}
// Updates title, trip_start_date, or trip_end_date on a draft trip.
// Only allowed while status = 'draft'. Returns the updated trip.
// ─────────────────────────────────────────────

func handlePatchTrip(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	tripPlanID := req.PathParameters["id"]
	if _, err := uuid.Parse(tripPlanID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid trip id"), nil
	}

	// Only the owner may patch
	ownerUserID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body PatchTripRequest
	if req.Body == "" {
		return jsonError(http.StatusBadRequest, "request body is required"), nil
	}
	if jsonErr := json.Unmarshal([]byte(req.Body), &body); jsonErr != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}

	// Verify trip exists, is owned by caller, and is in draft
	var currentStatus, currentTitle string
	var currentOwner string
	err = db.QueryRowContext(ctx,
		`SELECT status, title, owner_user_id::text FROM trip_plans WHERE id = $1`, tripPlanID,
	).Scan(&currentStatus, &currentTitle, &currentOwner)
	if err == sql.ErrNoRows {
		return jsonError(http.StatusNotFound, "trip not found"), nil
	}
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to load trip"), nil
	}
	if currentOwner != ownerUserID.String() {
		return jsonError(http.StatusForbidden, "only the trip owner may update trip details"), nil
	}
	if currentStatus != "draft" {
		return jsonError(http.StatusConflict, "trip details can only be updated while in draft status"), nil
	}

	// Apply patches
	newTitle := currentTitle
	if body.Title != nil && *body.Title != "" {
		newTitle = *body.Title
	}

	_, err = db.ExecContext(ctx, `
		UPDATE trip_plans
		SET title          = $2,
		    trip_start_date = $3,
		    trip_end_date   = $4,
		    updated_at      = NOW()
		WHERE id = $1
	`, tripPlanID, newTitle,
		optionalStr(body.TripStartDate),
		optionalStr(body.TripEndDate),
	)
	if err != nil {
		log.Printf("handlePatchTrip update: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to update trip"), nil
	}

	patchedStart := ""
	patchedEnd := ""
	if body.TripStartDate != nil {
		patchedStart = *body.TripStartDate
	}
	if body.TripEndDate != nil {
		patchedEnd = *body.TripEndDate
	}

	return jsonOK(CreateTripResponse{
		TripPlanID:    tripPlanID,
		Status:        "draft",
		Title:         newTitle,
		TripStartDate: patchedStart,
		TripEndDate:   patchedEnd,
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// optionalStr converts a *string pointer to an interface{} suitable for SQL
// parameterisation — nil pointer becomes SQL NULL.
func optionalStr(s *string) interface{} {
	if s == nil || *s == "" {
		return nil
	}
	return *s
}

func getTripStatus(ctx context.Context, tripPlanID string) (string, error) {
	var status string
	err := db.QueryRowContext(ctx,
		`SELECT status FROM trip_plans WHERE id = $1`, tripPlanID,
	).Scan(&status)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return status, err
}

// subFromRequest extracts the caller's UUID from the JWT authorizer context.
func subFromRequest(req events.APIGatewayProxyRequest) (uuid.UUID, error) {
	sub, ok := req.RequestContext.Authorizer["sub"].(string)
	if !ok || sub == "" {
		return uuid.Nil, fmt.Errorf("sub claim missing")
	}
	id, err := uuid.Parse(sub)
	if err != nil {
		return uuid.Nil, fmt.Errorf("sub claim is not a valid UUID: %w", err)
	}
	return id, nil
}

func scopeIn(scope string, scopes []string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

func jsonOK(v any) (events.APIGatewayProxyResponse, error) {
	b, _ := json.Marshal(v)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}, nil
}

func jsonError(code int, msg string) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func main() {
	lambda.Start(handler)
}
