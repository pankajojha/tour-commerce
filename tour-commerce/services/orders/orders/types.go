package orders

import "time"

// Order mirrors the commerce_orders table.
type Order struct {
	ID                string      `json:"id"`
	CheckoutSessionID *string     `json:"checkout_session_id,omitempty"`
	CartID            string      `json:"cart_id"`
	UserID            string      `json:"user_id"`
	TripPlanID        *string     `json:"trip_plan_id,omitempty"`
	Status            string      `json:"status"`
	TotalAmount       float64     `json:"total_amount"`
	RefundedAmount    float64     `json:"refunded_amount"`
	Currency          string      `json:"currency"`
	PaymentIntentID   *string     `json:"payment_intent_id,omitempty"`
	BillingDetails    any         `json:"billing_details"`
	Metadata          any         `json:"metadata"`
	Items             []OrderItem `json:"items,omitempty"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// OrderItem mirrors the commerce_order_items table.
type OrderItem struct {
	ID                   string  `json:"id"`
	OrderID              string  `json:"order_id"`
	ItemType             string  `json:"item_type"`
	ItemID               string  `json:"item_id"`
	TourAgentBookingID   *string `json:"tour_agent_booking_id,omitempty"`
	Quantity             int     `json:"quantity"`
	UnitPrice            float64 `json:"unit_price"`
	TotalPrice           float64 `json:"total_price"`
	ItemSnapshot         any     `json:"item_snapshot"`
	Status               string  `json:"status"`
	FailureReason        *string `json:"failure_reason,omitempty"`
	TravelerDate         *string `json:"traveler_date,omitempty"`
	CreatedAt            string  `json:"created_at"`
}

// CancelRequest is the body for POST /commerce/orders/{id}/cancel.
type CancelRequest struct {
	Reason string `json:"reason,omitempty"`
}

// ListOrdersResponse wraps a paginated list of orders.
type ListOrdersResponse struct {
	Orders []Order `json:"orders"`
	Total  int     `json:"total"`
}
