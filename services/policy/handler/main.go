package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	policydb "github.com/explorer-agent/policy/db"
	"github.com/explorer-agent/policy/evaluate"
	"github.com/explorer-agent/policy/rbac"
)

var db *sql.DB

func init() {
	var err error
	db, err = policydb.Open()
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
}

// ─────────────────────────────────────────────
// Request / response shapes
// ─────────────────────────────────────────────

type EvaluateRequest = evaluate.Request

type PolicyRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Scope          string `json:"scope"` // company|department|traveler_tier
	DepartmentID   string `json:"department_id,omitempty"`
	EffectiveFrom  string `json:"effective_from"` // ISO date
	EffectiveUntil string `json:"effective_until,omitempty"`
}

type RuleRequest struct {
	Category         string          `json:"category"`
	RuleKey          string          `json:"rule_key"`
	RuleValue        json.RawMessage `json:"rule_value"`
	EnforcementLevel string          `json:"enforcement_level"` // hard|soft|advisory
}

type OverrideRequest struct {
	EvaluationID string `json:"evaluation_id"`
	TripID       string `json:"trip_id"`
	ReasonCode   string `json:"reason_code"`
	ReasonNote   string `json:"reason_note,omitempty"`
}

type OverrideDecision struct {
	Status     string `json:"status"` // approved|denied
	ReasonNote string `json:"reason_note,omitempty"`
}

type RoleAssignRequest struct {
	UserID       string `json:"user_id"`
	RoleName     string `json:"role_name"`
	CompanyID    string `json:"company_id,omitempty"`
	DepartmentID string `json:"department_id,omitempty"`
}

// ─────────────────────────────────────────────
// Router
// ─────────────────────────────────────────────

type ServiceHandler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	routes := map[string]ServiceHandler{
		// Policy evaluation
		"POST /policy/evaluate": handleEvaluate,
		"GET /policy/limits":    handleGetLimits,

		// Policy CRUD (travel_manager+)
		"GET /policies":                handleListPolicies,
		"POST /policies":               handleCreatePolicy,
		"PATCH /policies/{id}":         handleUpdatePolicy,
		"DELETE /policies/{id}":        handleDeletePolicy,
		"POST /policies/{id}/activate": handleActivatePolicy,

		// Rules (attached to a policy)
		"POST /policies/{id}/rules":          handleAddRule,
		"DELETE /policies/{id}/rules/{rule}": handleDeleteRule,

		// Overrides
		"GET /policy/overrides":        handleListOverrides,
		"POST /policy/overrides":       handleRequestOverride,
		"PATCH /policy/overrides/{id}": handleDecideOverride,

		// Reports (dept_manager+)
		"GET /policy/reports/compliance": handleComplianceReport,

		// Role management (company_admin+)
		"GET /roles/users":   handleListUserRoles,
		"POST /roles/assign": handleAssignRole,
		"DELETE /roles/{id}": handleRevokeRole,
	}

	key := matchRoute(req.HTTPMethod, req.Path)
	if h, ok := routes[key]; ok {
		return h(ctx, req)
	}
	return jsonError(http.StatusNotFound, "route not found"), nil
}

func matchRoute(m, path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	n := len(parts)
	switch {
	// evaluation
	case m == "POST" && path == "/policy/evaluate":
		return "POST /policy/evaluate"
	case m == "GET" && path == "/policy/limits":
		return "GET /policy/limits"
	// policies
	case m == "GET" && path == "/policies":
		return "GET /policies"
	case m == "POST" && path == "/policies":
		return "POST /policies"
	case m == "PATCH" && n == 2 && parts[0] == "policies":
		return "PATCH /policies/{id}"
	case m == "DELETE" && n == 2 && parts[0] == "policies":
		return "DELETE /policies/{id}"
	case m == "POST" && n == 3 && parts[0] == "policies" && parts[2] == "activate":
		return "POST /policies/{id}/activate"
	case m == "POST" && n == 3 && parts[0] == "policies" && parts[2] == "rules":
		return "POST /policies/{id}/rules"
	case m == "DELETE" && n == 4 && parts[0] == "policies" && parts[2] == "rules":
		return "DELETE /policies/{id}/rules/{rule}"
	// overrides
	case m == "GET" && path == "/policy/overrides":
		return "GET /policy/overrides"
	case m == "POST" && path == "/policy/overrides":
		return "POST /policy/overrides"
	case m == "PATCH" && n == 3 && parts[0] == "policy" && parts[1] == "overrides":
		return "PATCH /policy/overrides/{id}"
	// reports
	case m == "GET" && n == 3 && parts[0] == "policy" && parts[1] == "reports":
		return "GET /policy/reports/compliance"
	// roles
	case m == "GET" && path == "/roles/users":
		return "GET /roles/users"
	case m == "POST" && path == "/roles/assign":
		return "POST /roles/assign"
	case m == "DELETE" && n == 2 && parts[0] == "roles":
		return "DELETE /roles/{id}"
	}
	return ""
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func callerID(req events.APIGatewayProxyRequest) string {
	if v, ok := req.RequestContext.Authorizer["sub"].(string); ok {
		return v
	}
	return ""
}

func pathID(req events.APIGatewayProxyRequest) string {
	parts := strings.Split(strings.Trim(req.Path, "/"), "/")
	for i, p := range parts {
		if (p == "policies" || p == "overrides" || p == "roles") && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func jsonOK(body interface{}) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(body)
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func jsonError(code int, msg string) events.APIGatewayProxyResponse {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return events.APIGatewayProxyResponse{
		StatusCode: code,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       string(b),
	}
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// ─────────────────────────────────────────────
// POST /policy/evaluate
// ─────────────────────────────────────────────

func handleEvaluate(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var body EvaluateRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.UserID == "" || body.TripID == "" || body.CompanyID == "" || body.Category == "" {
		return jsonError(http.StatusBadRequest, "user_id, trip_id, company_id, and category are required"), nil
	}

	result, err := evaluate.Evaluate(ctx, db, body)
	if err != nil {
		log.Printf("evaluate error: %v", err)
		return jsonError(http.StatusInternalServerError, "policy evaluation failed"), nil
	}
	return jsonOK(result), nil
}

// ─────────────────────────────────────────────
// GET /policy/limits?company_id=&category=&city=
// ─────────────────────────────────────────────

func handleGetLimits(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	companyID := req.QueryStringParameters["company_id"]
	category := req.QueryStringParameters["category"]
	city := req.QueryStringParameters["city"]

	if companyID == "" || category == "" {
		return jsonError(http.StatusBadRequest, "company_id and category are required"), nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT pr.rule_key, pr.rule_value, pr.enforcement_level, cp.version
		FROM policy_rules pr
		JOIN company_policies cp ON cp.id = pr.policy_id
		WHERE cp.company_id = $1::uuid
		  AND pr.category = $2
		  AND cp.is_active = TRUE
		  AND cp.effective_from <= CURRENT_DATE
		  AND (cp.effective_until IS NULL OR cp.effective_until >= CURRENT_DATE)
		ORDER BY cp.version DESC`,
		companyID, category,
	)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to load limits"), nil
	}
	defer rows.Close()

	limits := map[string]interface{}{}
	version := 0
	for rows.Next() {
		var ruleKey, enforcement string
		var ruleValue json.RawMessage
		if err := rows.Scan(&ruleKey, &ruleValue, &enforcement, &version); err != nil {
			continue
		}
		var rv map[string]interface{}
		_ = json.Unmarshal(ruleValue, &rv)

		// Apply city-tier override for hotel max_nightly_rate
		if category == "hotel" && ruleKey == "max_nightly_rate" && city != "" {
			if overrides, ok := rv["city_tier_overrides"].(map[string]interface{}); ok {
				if cityVal, ok := overrides[city]; ok {
					rv["max_nightly_rate"] = cityVal
				}
			}
		}
		for k, v := range rv {
			limits[k] = v
		}
	}

	return jsonOK(map[string]interface{}{
		"category":       category,
		"city":           city,
		"limits":         limits,
		"policy_version": version,
	}), nil
}

// ─────────────────────────────────────────────
// GET /policies
// ─────────────────────────────────────────────

func handleListPolicies(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	companyID := req.QueryStringParameters["company_id"]
	if companyID == "" {
		return jsonError(http.StatusBadRequest, "company_id is required"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:view", companyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id::text, name, description, scope, effective_from::text,
		       COALESCE(effective_until::text,''), is_active, version
		FROM company_policies
		WHERE company_id = $1::uuid
		ORDER BY is_active DESC, version DESC`, companyID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to list policies"), nil
	}
	defer rows.Close()

	type PolicyRow struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		Scope          string `json:"scope"`
		EffectiveFrom  string `json:"effective_from"`
		EffectiveUntil string `json:"effective_until,omitempty"`
		IsActive       bool   `json:"is_active"`
		Version        int    `json:"version"`
	}
	var policies []PolicyRow
	for rows.Next() {
		var p PolicyRow
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Scope,
			&p.EffectiveFrom, &p.EffectiveUntil, &p.IsActive, &p.Version); err != nil {
			continue
		}
		policies = append(policies, p)
	}
	if policies == nil {
		policies = []PolicyRow{}
	}
	return jsonOK(map[string]interface{}{"policies": policies}), nil
}

// ─────────────────────────────────────────────
// POST /policies
// ─────────────────────────────────────────────

func handleCreatePolicy(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	var body struct {
		PolicyRequest
		CompanyID string `json:"company_id"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.CompanyID == "" || body.Name == "" || body.EffectiveFrom == "" {
		return jsonError(http.StatusBadRequest, "company_id, name, and effective_from are required"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:create", body.CompanyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	scope := body.Scope
	if scope == "" {
		scope = "company"
	}
	var id string
	err = db.QueryRowContext(ctx, `
		INSERT INTO company_policies
			(company_id, name, description, scope, department_id,
			 effective_from, effective_until, created_by, updated_by)
		VALUES ($1::uuid,$2,$3,$4,$5,$6::date,$7,$8::uuid,$8::uuid)
		RETURNING id::text`,
		body.CompanyID, body.Name, body.Description, scope,
		nullStr(body.DepartmentID), body.EffectiveFrom,
		nullStr(body.EffectiveUntil), callerID,
	).Scan(&id)
	if err != nil {
		log.Printf("createPolicy error: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to create policy"), nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"policy_id":%q}`, id),
	}, nil
}

// ─────────────────────────────────────────────
// PATCH /policies/{id}
// ─────────────────────────────────────────────

func handleUpdatePolicy(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	policyID := pathID(req)
	var body PolicyRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}

	companyID, err := policyCompanyID(ctx, policyID)
	if err != nil {
		return jsonError(http.StatusNotFound, "policy not found"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:edit", companyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	_, err = db.ExecContext(ctx, `
		UPDATE company_policies
		SET name = COALESCE(NULLIF($2,''), name),
		    description = COALESCE(NULLIF($3,''), description),
		    effective_until = COALESCE(NULLIF($4,'')::date, effective_until),
		    updated_by = $5::uuid,
		    updated_at = NOW()
		WHERE id = $1::uuid`,
		policyID, body.Name, body.Description, body.EffectiveUntil, callerID,
	)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to update policy"), nil
	}
	return jsonOK(map[string]string{"policy_id": policyID, "status": "updated"}), nil
}

// ─────────────────────────────────────────────
// DELETE /policies/{id}
// ─────────────────────────────────────────────

func handleDeletePolicy(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	policyID := pathID(req)

	companyID, err := policyCompanyID(ctx, policyID)
	if err != nil {
		return jsonError(http.StatusNotFound, "policy not found"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:delete", companyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	_, err = db.ExecContext(ctx, `DELETE FROM company_policies WHERE id = $1::uuid`, policyID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to delete policy"), nil
	}
	return jsonOK(map[string]string{"status": "deleted"}), nil
}

// ─────────────────────────────────────────────
// POST /policies/{id}/activate
// ─────────────────────────────────────────────

func handleActivatePolicy(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	policyID := pathID(req)

	companyID, err := policyCompanyID(ctx, policyID)
	if err != nil {
		return jsonError(http.StatusNotFound, "policy not found"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:activate", companyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "transaction error"), nil
	}
	defer tx.Rollback() //nolint:errcheck

	// Deactivate all other policies for the same company + category scope
	_, err = tx.ExecContext(ctx,
		`UPDATE company_policies SET is_active=FALSE WHERE company_id=(
			SELECT company_id FROM company_policies WHERE id=$1::uuid) AND id<>$1::uuid`,
		policyID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to deactivate previous policies"), nil
	}

	var newVersion int
	err = tx.QueryRowContext(ctx,
		`UPDATE company_policies SET is_active=TRUE, version=version+1, updated_by=$2::uuid, updated_at=NOW()
		 WHERE id=$1::uuid RETURNING version`,
		policyID, callerID).Scan(&newVersion)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to activate policy"), nil
	}

	if err = tx.Commit(); err != nil {
		return jsonError(http.StatusInternalServerError, "commit failed"), nil
	}
	return jsonOK(map[string]interface{}{"policy_id": policyID, "is_active": true, "version": newVersion}), nil
}

// ─────────────────────────────────────────────
// POST /policies/{id}/rules
// ─────────────────────────────────────────────

func handleAddRule(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	policyID := pathID(req)
	var body RuleRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}

	companyID, err := policyCompanyID(ctx, policyID)
	if err != nil {
		return jsonError(http.StatusNotFound, "policy not found"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:edit", companyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	enforcement := body.EnforcementLevel
	if enforcement == "" {
		enforcement = "hard"
	}
	var ruleID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO policy_rules (policy_id, category, rule_key, rule_value, enforcement_level)
		VALUES ($1::uuid, $2, $3, $4::jsonb, $5)
		ON CONFLICT (policy_id, category, rule_key) DO UPDATE
		  SET rule_value = EXCLUDED.rule_value, enforcement_level = EXCLUDED.enforcement_level
		RETURNING id::text`,
		policyID, body.Category, body.RuleKey, string(body.RuleValue), enforcement,
	).Scan(&ruleID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to add rule"), nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"rule_id":%q}`, ruleID),
	}, nil
}

// ─────────────────────────────────────────────
// DELETE /policies/{id}/rules/{rule}
// ─────────────────────────────────────────────

func handleDeleteRule(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	parts := strings.Split(strings.Trim(req.Path, "/"), "/")
	policyID := parts[1]
	ruleID := parts[3]

	companyID, err := policyCompanyID(ctx, policyID)
	if err != nil {
		return jsonError(http.StatusNotFound, "policy not found"), nil
	}
	ok, err := rbac.HasPermission(ctx, db, callerID, "policy:edit", companyID)
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	_, err = db.ExecContext(ctx,
		`DELETE FROM policy_rules WHERE id=$1::uuid AND policy_id=$2::uuid`, ruleID, policyID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to delete rule"), nil
	}
	return jsonOK(map[string]string{"status": "deleted"}), nil
}

// ─────────────────────────────────────────────
// GET /policy/overrides
// ─────────────────────────────────────────────

func handleListOverrides(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	statusFilter := req.QueryStringParameters["status"] // optional

	canApprove, _ := rbac.HasPermission(ctx, db, callerID, "override:approve", "")
	var rows *sql.Rows
	var err error
	if canApprove {
		if statusFilter != "" {
			rows, err = db.QueryContext(ctx,
				`SELECT id::text, evaluation_id::text, trip_id::text, requester_user_id::text,
				        reason_code, COALESCE(reason_note,''), status, requested_at::text,
				        COALESCE(sla_deadline::text,'')
				 FROM policy_overrides WHERE status=$1 ORDER BY requested_at`, statusFilter)
		} else {
			rows, err = db.QueryContext(ctx,
				`SELECT id::text, evaluation_id::text, trip_id::text, requester_user_id::text,
				        reason_code, COALESCE(reason_note,''), status, requested_at::text,
				        COALESCE(sla_deadline::text,'')
				 FROM policy_overrides ORDER BY requested_at DESC`)
		}
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT id::text, evaluation_id::text, trip_id::text, requester_user_id::text,
			        reason_code, COALESCE(reason_note,''), status, requested_at::text,
			        COALESCE(sla_deadline::text,'')
			 FROM policy_overrides WHERE requester_user_id=$1::uuid ORDER BY requested_at DESC`, callerID)
	}
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to list overrides"), nil
	}
	defer rows.Close()

	type OverrideRow struct {
		ID           string `json:"id"`
		EvaluationID string `json:"evaluation_id"`
		TripID       string `json:"trip_id"`
		RequesterID  string `json:"requester_user_id"`
		ReasonCode   string `json:"reason_code"`
		ReasonNote   string `json:"reason_note,omitempty"`
		Status       string `json:"status"`
		RequestedAt  string `json:"requested_at"`
		SLADeadline  string `json:"sla_deadline,omitempty"`
	}
	var overrides []OverrideRow
	for rows.Next() {
		var o OverrideRow
		if err := rows.Scan(&o.ID, &o.EvaluationID, &o.TripID, &o.RequesterID,
			&o.ReasonCode, &o.ReasonNote, &o.Status, &o.RequestedAt, &o.SLADeadline); err != nil {
			continue
		}
		overrides = append(overrides, o)
	}
	if overrides == nil {
		overrides = []OverrideRow{}
	}
	return jsonOK(map[string]interface{}{"overrides": overrides}), nil
}

// ─────────────────────────────────────────────
// POST /policy/overrides
// ─────────────────────────────────────────────

func handleRequestOverride(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	var body OverrideRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.EvaluationID == "" || body.TripID == "" || body.ReasonCode == "" {
		return jsonError(http.StatusBadRequest, "evaluation_id, trip_id, and reason_code are required"), nil
	}

	var id string
	err := db.QueryRowContext(ctx, `
		INSERT INTO policy_overrides (evaluation_id, trip_id, requester_user_id, reason_code, reason_note)
		VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
		RETURNING id::text`,
		body.EvaluationID, body.TripID, callerID, body.ReasonCode, nullStr(body.ReasonNote),
	).Scan(&id)
	if err != nil {
		log.Printf("requestOverride error: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to create override request"), nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"override_id":%q,"status":"pending"}`, id),
	}, nil
}

// ─────────────────────────────────────────────
// PATCH /policy/overrides/{id}
// ─────────────────────────────────────────────

func handleDecideOverride(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	ok, err := rbac.HasPermission(ctx, db, callerID, "override:approve", "")
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	overrideID := pathID(req)
	var body OverrideDecision
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.Status != "approved" && body.Status != "denied" {
		return jsonError(http.StatusBadRequest, "status must be 'approved' or 'denied'"), nil
	}

	_, err = db.ExecContext(ctx, `
		UPDATE policy_overrides
		SET status=$2, approver_user_id=$3::uuid, resolved_at=$4,
		    reason_note=COALESCE(NULLIF($5,''), reason_note)
		WHERE id=$1::uuid AND status='pending'`,
		overrideID, body.Status, callerID, time.Now().UTC(), body.ReasonNote,
	)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to update override"), nil
	}
	return jsonOK(map[string]string{"override_id": overrideID, "status": body.Status}), nil
}

// ─────────────────────────────────────────────
// GET /policy/reports/compliance
// ─────────────────────────────────────────────

func handleComplianceReport(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	ok, err := rbac.HasPermission(ctx, db, callerID, "report:view", "")
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	type CategorySummary struct {
		Category string `json:"category"`
		Result   string `json:"result"`
		Count    int    `json:"count"`
	}
	type TopViolation struct {
		RuleKey   string `json:"rule_key"`
		Frequency int    `json:"frequency"`
	}
	type OverrideSummary struct {
		Approved int `json:"approved"`
		Denied   int `json:"denied"`
		Pending  int `json:"pending"`
	}

	rows, err := db.QueryContext(ctx, `
		SELECT category, result, COUNT(*)::int
		FROM policy_evaluations
		WHERE evaluated_at >= date_trunc('month', now())
		GROUP BY category, result ORDER BY category, result`)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "report query failed"), nil
	}
	defer rows.Close()
	var compliance []CategorySummary
	for rows.Next() {
		var c CategorySummary
		_ = rows.Scan(&c.Category, &c.Result, &c.Count)
		compliance = append(compliance, c)
	}

	vrows, err := db.QueryContext(ctx, `
		SELECT violations->0->>'rule_key' AS rule, COUNT(*)::int as freq
		FROM policy_evaluations
		WHERE result='non_compliant' AND evaluated_at >= now() - interval '30 days'
		  AND violations <> '[]'
		GROUP BY rule ORDER BY freq DESC LIMIT 10`)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "violations query failed"), nil
	}
	defer vrows.Close()
	var topViolations []TopViolation
	for vrows.Next() {
		var v TopViolation
		_ = vrows.Scan(&v.RuleKey, &v.Frequency)
		topViolations = append(topViolations, v)
	}

	var ovr OverrideSummary
	_ = db.QueryRowContext(ctx, `
		SELECT
		  COUNT(*) FILTER (WHERE status='approved'),
		  COUNT(*) FILTER (WHERE status='denied'),
		  COUNT(*) FILTER (WHERE status='pending')
		FROM policy_overrides
		WHERE requested_at >= now() - interval '30 days'`).
		Scan(&ovr.Approved, &ovr.Denied, &ovr.Pending)

	return jsonOK(map[string]interface{}{
		"period":         "current_month",
		"compliance":     compliance,
		"top_violations": topViolations,
		"overrides_30d":  ovr,
	}), nil
}

// ─────────────────────────────────────────────
// GET /roles/users
// ─────────────────────────────────────────────

func handleListUserRoles(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	ok, err := rbac.HasPermission(ctx, db, callerID, "role:view", "")
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}
	companyID := req.QueryStringParameters["company_id"]

	var rows *sql.Rows
	if companyID != "" {
		rows, err = db.QueryContext(ctx,
			`SELECT ur.id::text, ur.user_id::text, ur.role_name, r.level,
			        COALESCE(ur.company_id::text,''), ur.assigned_at::text
			 FROM user_roles ur JOIN roles r ON r.name=ur.role_name
			 WHERE ur.company_id=$1::uuid ORDER BY r.level`, companyID)
	} else {
		rows, err = db.QueryContext(ctx,
			`SELECT ur.id::text, ur.user_id::text, ur.role_name, r.level,
			        COALESCE(ur.company_id::text,''), ur.assigned_at::text
			 FROM user_roles ur JOIN roles r ON r.name=ur.role_name ORDER BY r.level`)
	}
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to list user roles"), nil
	}
	defer rows.Close()

	type UserRoleRow struct {
		ID         string `json:"id"`
		UserID     string `json:"user_id"`
		RoleName   string `json:"role_name"`
		Level      int    `json:"level"`
		CompanyID  string `json:"company_id,omitempty"`
		AssignedAt string `json:"assigned_at"`
	}
	var userRoles []UserRoleRow
	for rows.Next() {
		var r UserRoleRow
		_ = rows.Scan(&r.ID, &r.UserID, &r.RoleName, &r.Level, &r.CompanyID, &r.AssignedAt)
		userRoles = append(userRoles, r)
	}
	if userRoles == nil {
		userRoles = []UserRoleRow{}
	}
	return jsonOK(map[string]interface{}{"user_roles": userRoles}), nil
}

// ─────────────────────────────────────────────
// POST /roles/assign
// ─────────────────────────────────────────────

func handleAssignRole(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	ok, err := rbac.HasPermission(ctx, db, callerID, "role:assign", "")
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}

	var body RoleAssignRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.UserID == "" || body.RoleName == "" {
		return jsonError(http.StatusBadRequest, "user_id and role_name are required"), nil
	}

	// Prevent privilege escalation: caller cannot assign a role with higher privilege than their own
	callerLevel, _ := rbac.RoleLevel(ctx, db, callerID, body.CompanyID)
	var targetLevel int
	if err := db.QueryRowContext(ctx, `SELECT level FROM roles WHERE name=$1`, body.RoleName).Scan(&targetLevel); err != nil {
		return jsonError(http.StatusBadRequest, "unknown role name"), nil
	}
	if targetLevel < callerLevel {
		return jsonError(http.StatusForbidden, "cannot assign a role with higher privilege than your own"), nil
	}

	var id string
	err = db.QueryRowContext(ctx, `
		INSERT INTO user_roles (user_id, role_name, company_id, department_id, assigned_by)
		VALUES ($1::uuid, $2, $3, $4, $5::uuid)
		ON CONFLICT (user_id, company_id, role_name) DO UPDATE SET assigned_by=EXCLUDED.assigned_by
		RETURNING id::text`,
		body.UserID, body.RoleName, nullStr(body.CompanyID),
		nullStr(body.DepartmentID), callerID,
	).Scan(&id)
	if err != nil {
		log.Printf("assignRole error: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to assign role"), nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       fmt.Sprintf(`{"assignment_id":%q}`, id),
	}, nil
}

// ─────────────────────────────────────────────
// DELETE /roles/{id}
// ─────────────────────────────────────────────

func handleRevokeRole(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	callerID := callerID(req)
	ok, err := rbac.HasPermission(ctx, db, callerID, "role:assign", "")
	if err != nil || !ok {
		return jsonError(http.StatusForbidden, "insufficient permissions"), nil
	}
	assignmentID := pathID(req)
	_, err = db.ExecContext(ctx, `DELETE FROM user_roles WHERE id=$1::uuid`, assignmentID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to revoke role"), nil
	}
	return jsonOK(map[string]string{"status": "revoked"}), nil
}

// ─────────────────────────────────────────────
// Shared helpers
// ─────────────────────────────────────────────

// policyCompanyID fetches the company_id for a policy (used for RBAC checks).
func policyCompanyID(ctx context.Context, policyID string) (string, error) {
	var companyID string
	err := db.QueryRowContext(ctx,
		`SELECT company_id::text FROM company_policies WHERE id=$1::uuid`, policyID,
	).Scan(&companyID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("not found")
	}
	return companyID, err
}

// ─────────────────────────────────────────────
// Entry point
// ─────────────────────────────────────────────

func main() {
	_ = uuid.New() // ensure import is used
	lambda.Start(handler)
}
