package checkout

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Open returns a connected *sql.DB using DATABASE_URL from the environment.
func Open() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	return db, db.Ping()
}

// CartSummary is the minimal cart info needed for checkout.
type CartSummary struct {
	ID          string
	UserID      string
	TripPlanID  string
	Status      string
	Currency    string
	TotalAmount float64
	Items       []CartItemSummary
}

// CartItemSummary is the minimal item info needed to create bookings.
type CartItemSummary struct {
	ID           string
	ItemType     string
	ItemID       string
	Quantity     int
	UnitPrice    float64
	TotalPrice   float64
	ItemSnapshot map[string]any
	TravelerDate string
}

// GetCartSummary loads the cart and all its items for checkout validation.
func GetCartSummary(ctx context.Context, db *sql.DB, cartID, userID string) (*CartSummary, error) {
	cs := &CartSummary{}
	var tripPlanID sql.NullString

	err := db.QueryRowContext(ctx, `
		SELECT id, user_id, trip_plan_id, status, currency
		FROM commerce_carts WHERE id = $1 AND user_id = $2
	`, cartID, userID).Scan(&cs.ID, &cs.UserID, &tripPlanID, &cs.Status, &cs.Currency)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetCartSummary: %w", err)
	}
	if tripPlanID.Valid {
		cs.TripPlanID = tripPlanID.String
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, item_type, item_id, quantity, unit_price, total_price, item_snapshot, traveler_date
		FROM commerce_cart_items WHERE cart_id = $1
	`, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item CartItemSummary
		var snapshotRaw []byte
		var travelerDate sql.NullString
		if err = rows.Scan(&item.ID, &item.ItemType, &item.ItemID,
			&item.Quantity, &item.UnitPrice, &item.TotalPrice,
			&snapshotRaw, &travelerDate,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(snapshotRaw, &item.ItemSnapshot)
		if travelerDate.Valid {
			item.TravelerDate = travelerDate.String
		}
		cs.TotalAmount += item.TotalPrice
		cs.Items = append(cs.Items, item)
	}
	return cs, nil
}

// CreateSession inserts a checkout session and returns it.
func CreateSession(ctx context.Context, db *sql.DB, cartID, userID string, totalAmount float64, currency string, billing any) (*Session, error) {
	billingRaw, _ := json.Marshal(billing)
	expires := time.Now().UTC().Add(30 * time.Minute)

	s := &Session{}
	err := db.QueryRowContext(ctx, `
		INSERT INTO commerce_checkout_sessions (cart_id, user_id, total_amount, currency, billing_details, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, cart_id, user_id, status, billing_details, total_amount, currency, expires_at, created_at, updated_at
	`, cartID, userID, totalAmount, currency, billingRaw, expires).Scan(
		&s.ID, &s.CartID, &s.UserID, &s.Status,
		new([]byte), &s.TotalAmount, &s.Currency, &s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateSession: %w", err)
	}
	s.BillingDetails = billing
	return s, nil
}

// GetSession retrieves a checkout session by ID.
func GetSession(ctx context.Context, db *sql.DB, sessionID string) (*Session, error) {
	s := &Session{}
	var billingRaw, aiRaw []byte
	err := db.QueryRowContext(ctx, `
		SELECT id, cart_id, user_id, status, ai_validation, billing_details, total_amount, currency, expires_at, created_at, updated_at
		FROM commerce_checkout_sessions WHERE id = $1
	`, sessionID).Scan(
		&s.ID, &s.CartID, &s.UserID, &s.Status,
		&aiRaw, &billingRaw, &s.TotalAmount, &s.Currency,
		&s.ExpiresAt, &s.CreatedAt, &s.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSession: %w", err)
	}
	_ = json.Unmarshal(billingRaw, &s.BillingDetails)
	_ = json.Unmarshal(aiRaw, &s.AIValidation)
	return s, nil
}

// UpdateSessionStatus sets the session status and optionally stores AI validation.
func UpdateSessionStatus(ctx context.Context, db *sql.DB, sessionID string, status SessionStatus, aiValidation any) error {
	aiRaw, _ := json.Marshal(aiValidation)
	_, err := db.ExecContext(ctx, `
		UPDATE commerce_checkout_sessions
		SET status = $2, ai_validation = $3
		WHERE id = $1
	`, sessionID, status, aiRaw)
	return err
}

// CreateOrder inserts an order from a confirmed checkout session.
func CreateOrder(ctx context.Context, db *sql.DB, session *Session, cart *CartSummary, paymentIntentID string) (string, error) {
	var orderID string
	err := db.QueryRowContext(ctx, `
		INSERT INTO commerce_orders
		    (checkout_session_id, cart_id, user_id, trip_plan_id, total_amount, currency, payment_intent_id, billing_details)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id
	`, session.ID, session.CartID, session.UserID,
		nullStr(cart.TripPlanID), session.TotalAmount, session.Currency,
		nullStr(paymentIntentID), mustJSON(session.BillingDetails),
	).Scan(&orderID)
	if err != nil {
		return "", fmt.Errorf("CreateOrder: %w", err)
	}

	for _, item := range cart.Items {
		snapshotRaw, _ := json.Marshal(item.ItemSnapshot)
		_, err = db.ExecContext(ctx, `
			INSERT INTO commerce_order_items
			    (order_id, item_type, item_id, quantity, unit_price, total_price, item_snapshot, traveler_date)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		`, orderID, item.ItemType, item.ItemID,
			item.Quantity, item.UnitPrice, item.TotalPrice,
			snapshotRaw, nullStr(item.TravelerDate),
		)
		if err != nil {
			return "", fmt.Errorf("CreateOrder insert item: %w", err)
		}
	}
	return orderID, nil
}

// ConfirmOrderItem updates a single order item with the tour-agent booking ID.
func ConfirmOrderItem(ctx context.Context, db *sql.DB, orderID, itemID, tourAgentBookingID string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE commerce_order_items
		SET status = 'confirmed', tour_agent_booking_id = $3
		WHERE order_id = $1 AND id = $2
	`, orderID, itemID, tourAgentBookingID)
	return err
}

// SetOrderStatus updates the top-level order status.
func SetOrderStatus(ctx context.Context, db *sql.DB, orderID, status string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE commerce_orders SET status = $2 WHERE id = $1`, orderID, status)
	return err
}

// LogAIInteraction persists an AI commerce interaction for audit.
func LogAIInteraction(ctx context.Context, db *sql.DB, userID, cartID, interactionType string, request, response any, agentIntent string) error {
	reqRaw, _ := json.Marshal(request)
	respRaw, _ := json.Marshal(response)
	_, err := db.ExecContext(ctx, `
		INSERT INTO commerce_ai_interactions
		    (user_id, cart_id, interaction_type, request_payload, response_payload, agent_intent)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, userID, nullStr(cartID), interactionType, reqRaw, respRaw, nullStr(agentIntent))
	return err
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
