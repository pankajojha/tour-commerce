package checkout

import "time"

// SessionStatus is the lifecycle state of a checkout session.
type SessionStatus string

const (
	SessionStatusPending          SessionStatus = "pending"
	SessionStatusAIValidated      SessionStatus = "ai_validated"
	SessionStatusPaymentInitiated SessionStatus = "payment_initiated"
	SessionStatusConfirmed        SessionStatus = "confirmed"
	SessionStatusFailed           SessionStatus = "failed"
	SessionStatusExpired          SessionStatus = "expired"
)

// Session is a commerce_checkout_sessions row.
type Session struct {
	ID             string        `json:"id"`
	CartID         string        `json:"cart_id"`
	UserID         string        `json:"user_id"`
	Status         SessionStatus `json:"status"`
	AIValidation   any           `json:"ai_validation,omitempty"`
	BillingDetails any           `json:"billing_details"`
	TotalAmount    float64       `json:"total_amount"`
	Currency       string        `json:"currency"`
	ExpiresAt      time.Time     `json:"expires_at"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// InitiateRequest is the body for POST /commerce/checkout.
type InitiateRequest struct {
	CartID         string `json:"cart_id"`
	BillingDetails any    `json:"billing_details"`
}

// ConfirmRequest is the body for POST /commerce/checkout/{session_id}/confirm.
type ConfirmRequest struct {
	PaymentIntentID string `json:"payment_intent_id"`
}

// AIValidationResult is the structure returned by tour-planner-ai for budget/conflict checks.
type AIValidationResult struct {
	Valid            bool     `json:"valid"`
	BudgetStatus     string   `json:"budget_status"`     // within_budget, over_budget, no_budget_set
	TotalCost        float64  `json:"total_cost"`
	ConflictsFound   bool     `json:"conflicts_found"`
	Conflicts        []string `json:"conflicts,omitempty"`
	Warnings         []string `json:"warnings,omitempty"`
	Recommendations  []string `json:"recommendations,omitempty"`
	AgentIntent      string   `json:"agent_intent"`
}

// TourAgentBookingRequest is sent to tour-agent to create a booking for one cart item.
type TourAgentBookingRequest struct {
	UserID       string  `json:"user_id"`
	ExcursionID  string  `json:"excursion_id"`
	TripPlanID   string  `json:"trip_plan_id,omitempty"`
	TravelerDate string  `json:"traveler_date,omitempty"`
	Quantity     int     `json:"quantity"`
	TotalAmount  float64 `json:"total_amount"`
}

// TourAgentBookingResponse is the minimal response shape we parse from tour-agent.
type TourAgentBookingResponse struct {
	BookingID string `json:"booking_id"`
	Status    string `json:"status"`
}
