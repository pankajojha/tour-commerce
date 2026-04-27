package evaluate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Input is the booking item to evaluate against corporate policy.
type Input struct {
	// Hotel
	NightlyRate float64 `json:"nightly_rate,omitempty"`
	City        string  `json:"city,omitempty"`
	Vendor      string  `json:"vendor,omitempty"`
	CheckIn     string  `json:"check_in,omitempty"`
	CheckOut    string  `json:"check_out,omitempty"`
	// Flight
	CabinClass         string `json:"cabin_class,omitempty"`
	AdvanceBookingDays int    `json:"advance_booking_days,omitempty"`
	// Car
	CarClass string `json:"car_class,omitempty"`
	// Meal
	MealType    string  `json:"meal_type,omitempty"` // breakfast|lunch|dinner
	MealAmount  float64 `json:"meal_amount,omitempty"`
	DailyAmount float64 `json:"daily_amount,omitempty"`
	// Common
	Currency string `json:"currency,omitempty"`
}

// Request is the full evaluation request.
type Request struct {
	UserID        string `json:"user_id"`
	TripID        string `json:"trip_id"`
	CompanyID     string `json:"company_id"`
	Category      string `json:"category"`
	Input         Input  `json:"input"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

// Violation describes a single policy breach.
type Violation struct {
	RuleKey     string      `json:"rule_key"`
	RuleValue   interface{} `json:"rule_value"`
	ActualValue interface{} `json:"actual_value"`
	Message     string      `json:"message"`
}

// Alternative is a compliant option returned alongside violations.
type Alternative struct {
	Vendor    string  `json:"vendor,omitempty"`
	Price     float64 `json:"price,omitempty"`
	Label     string  `json:"label,omitempty"`
	Compliant bool    `json:"compliant"`
}

// Result is the policy evaluation result.
type Result struct {
	Result           string        `json:"result"` // compliant|non_compliant|advisory
	Violations       []Violation   `json:"violations"`
	Alternatives     []Alternative `json:"allowed_alternatives"`
	PolicyVersion    int           `json:"policy_version"`
	OverrideRequired bool          `json:"override_required"`
	EvaluationID     string        `json:"evaluation_id"`
}

// PolicyRule is a row from policy_rules joined with its policy.
type policyRule struct {
	PolicyID      string
	PolicyVersion int
	RuleKey       string
	RuleValue     json.RawMessage
	Enforcement   string // hard|soft|advisory
}

// Evaluate runs all active policy rules for the company+category and returns a Result.
// It also persists the evaluation to policy_evaluations.
func Evaluate(ctx context.Context, db *sql.DB, req Request) (Result, error) {
	rules, policyID, version, err := loadRules(ctx, db, req.CompanyID, req.Category)
	if err != nil {
		return Result{}, fmt.Errorf("evaluate: load rules: %w", err)
	}

	violations := []Violation{}
	hardViolation := false
	advisoryOnly := true

	for _, rule := range rules {
		v, violated := checkRule(rule, req.Category, req.Input)
		if violated {
			violations = append(violations, v)
			switch rule.Enforcement {
			case "hard":
				hardViolation = true
				advisoryOnly = false
			case "soft":
				advisoryOnly = false
			}
		}
	}

	resultStr := "compliant"
	if hardViolation {
		resultStr = "non_compliant"
	} else if len(violations) > 0 && !advisoryOnly {
		resultStr = "non_compliant"
	} else if len(violations) > 0 && advisoryOnly {
		resultStr = "advisory"
	}

	inputJSON, _ := json.Marshal(req.Input)
	violationsJSON, _ := json.Marshal(violations)

	var evalID string
	err = db.QueryRowContext(ctx, `
		INSERT INTO policy_evaluations
			(trip_id, user_id, policy_id, category, input_snapshot, result,
			 violations, allowed_alternatives, policy_version, correlation_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
		RETURNING id::text`,
		req.TripID, req.UserID, nullStr(policyID), req.Category,
		string(inputJSON), resultStr,
		string(violationsJSON), "[]", version,
		nullStr(req.CorrelationID),
	).Scan(&evalID)
	if err != nil {
		return Result{}, fmt.Errorf("evaluate: insert evaluation: %w", err)
	}

	return Result{
		Result:           resultStr,
		Violations:       violations,
		Alternatives:     []Alternative{},
		PolicyVersion:    version,
		OverrideRequired: hardViolation,
		EvaluationID:     evalID,
	}, nil
}

// loadRules fetches all active policy rules for the given company and category.
func loadRules(ctx context.Context, db *sql.DB, companyID, category string) ([]policyRule, string, int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT pr.policy_id::text, cp.version, pr.rule_key, pr.rule_value, pr.enforcement_level
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
		return nil, "", 0, err
	}
	defer rows.Close()

	var rules []policyRule
	var policyID string
	var version int
	for rows.Next() {
		var r policyRule
		if err := rows.Scan(&r.PolicyID, &r.PolicyVersion, &r.RuleKey, &r.RuleValue, &r.Enforcement); err != nil {
			return nil, "", 0, err
		}
		if policyID == "" {
			policyID = r.PolicyID
			version = r.PolicyVersion
		}
		rules = append(rules, r)
	}
	return rules, policyID, version, rows.Err()
}

// checkRule evaluates a single rule against the input. Returns (violation, wasViolated).
func checkRule(rule policyRule, category string, input Input) (Violation, bool) {
	var rv map[string]interface{}
	_ = json.Unmarshal(rule.RuleValue, &rv)

	switch category {
	case "hotel":
		return checkHotelRule(rule, rv, input)
	case "flight":
		return checkFlightRule(rule, rv, input)
	case "car":
		return checkCarRule(rule, rv, input)
	case "meal":
		return checkMealRule(rule, rv, input)
	case "advance_booking":
		return checkAdvanceBookingRule(rule, rv, input)
	}
	return Violation{}, false
}

func checkHotelRule(rule policyRule, rv map[string]interface{}, input Input) (Violation, bool) {
	switch rule.RuleKey {
	case "max_nightly_rate":
		cap := jsonFloat(rv, "max_nightly_rate")
		// Check city-tier override
		if overrides, ok := rv["city_tier_overrides"].(map[string]interface{}); ok {
			if cityVal, ok := overrides[input.City]; ok {
				if cf, ok2 := toFloat(cityVal); ok2 {
					cap = cf
				}
			}
		}
		if cap > 0 && input.NightlyRate > cap {
			return Violation{
				RuleKey:     "max_nightly_rate",
				RuleValue:   cap,
				ActualValue: input.NightlyRate,
				Message:     fmt.Sprintf("Nightly rate %.2f exceeds cap of %.2f for %s", input.NightlyRate, cap, input.City),
			}, true
		}
	case "preferred_vendors":
		vendors := jsonStringSlice(rv, "preferred_vendors")
		if len(vendors) > 0 && input.Vendor != "" && !contains(vendors, input.Vendor) {
			return Violation{
				RuleKey:     "preferred_vendors",
				RuleValue:   vendors,
				ActualValue: input.Vendor,
				Message:     fmt.Sprintf("Vendor %q is not on the preferred vendor list", input.Vendor),
			}, true
		}
	}
	return Violation{}, false
}

func checkFlightRule(rule policyRule, rv map[string]interface{}, input Input) (Violation, bool) {
	cabinOrder := map[string]int{"economy": 1, "premium_economy": 2, "business": 3, "first": 4}
	switch rule.RuleKey {
	case "max_cabin_class":
		maxCabin := jsonString(rv, "max_cabin_class")
		if maxCabin != "" && input.CabinClass != "" {
			if cabinOrder[input.CabinClass] > cabinOrder[maxCabin] {
				return Violation{
					RuleKey:     "max_cabin_class",
					RuleValue:   maxCabin,
					ActualValue: input.CabinClass,
					Message:     fmt.Sprintf("Cabin class %q exceeds policy maximum of %q", input.CabinClass, maxCabin),
				}, true
			}
		}
	case "advance_booking_days":
		minDays := int(jsonFloat(rv, "advance_booking_days"))
		if minDays > 0 && input.AdvanceBookingDays < minDays {
			return Violation{
				RuleKey:     "advance_booking_days",
				RuleValue:   minDays,
				ActualValue: input.AdvanceBookingDays,
				Message:     fmt.Sprintf("Flight booked %d days in advance; policy requires %d days minimum", input.AdvanceBookingDays, minDays),
			}, true
		}
	}
	return Violation{}, false
}

var carClassOrder = map[string]int{"economy": 1, "compact": 2, "standard": 3, "full_size": 4, "premium": 5, "luxury": 6, "suv": 7}

func checkCarRule(rule policyRule, rv map[string]interface{}, input Input) (Violation, bool) {
	switch rule.RuleKey {
	case "max_car_class":
		maxClass := jsonString(rv, "max_car_class")
		if maxClass != "" && input.CarClass != "" {
			if carClassOrder[input.CarClass] > carClassOrder[maxClass] {
				return Violation{
					RuleKey:     "max_car_class",
					RuleValue:   maxClass,
					ActualValue: input.CarClass,
					Message:     fmt.Sprintf("Car class %q exceeds policy maximum of %q", input.CarClass, maxClass),
				}, true
			}
		}
	}
	return Violation{}, false
}

func checkMealRule(rule policyRule, rv map[string]interface{}, input Input) (Violation, bool) {
	switch rule.RuleKey {
	case "daily_allowance":
		cap := jsonFloat(rv, "daily_allowance")
		if cap > 0 && input.DailyAmount > cap {
			return Violation{
				RuleKey:     "daily_allowance",
				RuleValue:   cap,
				ActualValue: input.DailyAmount,
				Message:     fmt.Sprintf("Daily meal spend %.2f exceeds allowance of %.2f", input.DailyAmount, cap),
			}, true
		}
	case "per_meal_caps":
		if caps, ok := rv["per_meal_caps"].(map[string]interface{}); ok && input.MealType != "" {
			if capVal, ok := caps[input.MealType]; ok {
				if cf, ok2 := toFloat(capVal); ok2 && input.MealAmount > cf {
					return Violation{
						RuleKey:     "per_meal_caps",
						RuleValue:   cf,
						ActualValue: input.MealAmount,
						Message:     fmt.Sprintf("%s meal amount %.2f exceeds cap of %.2f", input.MealType, input.MealAmount, cf),
					}, true
				}
			}
		}
	}
	return Violation{}, false
}

func checkAdvanceBookingRule(rule policyRule, rv map[string]interface{}, input Input) (Violation, bool) {
	_ = time.Now() // keep time import used
	switch rule.RuleKey {
	case "min_advance_days":
		minDays := int(jsonFloat(rv, "min_advance_days"))
		if minDays > 0 && input.AdvanceBookingDays < minDays {
			return Violation{
				RuleKey:     "min_advance_days",
				RuleValue:   minDays,
				ActualValue: input.AdvanceBookingDays,
				Message:     fmt.Sprintf("Booking made %d days in advance; policy requires %d days minimum", input.AdvanceBookingDays, minDays),
			}, true
		}
	}
	return Violation{}, false
}

// ── helpers ───────────────────────────────────────────────────────────────────

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func jsonFloat(m map[string]interface{}, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	f, _ := toFloat(v)
	return f
}

func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

func jsonString(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func jsonStringSlice(m map[string]interface{}, key string) []string {
	v, ok := m[key]
	if !ok {
		return nil
	}
	raw, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
