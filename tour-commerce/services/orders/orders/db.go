package orders

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Open returns a connected *sql.DB using DATABASE_URL.
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

// GetOrder returns an order with all its items.
func GetOrder(ctx context.Context, db *sql.DB, orderID string) (*Order, error) {
	o := &Order{}
	var sessionID, tripPlanID, paymentIntentID sql.NullString
	var billingRaw, metaRaw []byte

	err := db.QueryRowContext(ctx, `
		SELECT id, checkout_session_id, cart_id, user_id, trip_plan_id, status,
		       total_amount, refunded_amount, currency, payment_intent_id,
		       billing_details, metadata, created_at, updated_at
		FROM commerce_orders WHERE id = $1
	`, orderID).Scan(
		&o.ID, &sessionID, &o.CartID, &o.UserID, &tripPlanID, &o.Status,
		&o.TotalAmount, &o.RefundedAmount, &o.Currency, &paymentIntentID,
		&billingRaw, &metaRaw, &o.CreatedAt, &o.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetOrder: %w", err)
	}
	if sessionID.Valid {
		o.CheckoutSessionID = &sessionID.String
	}
	if tripPlanID.Valid {
		o.TripPlanID = &tripPlanID.String
	}
	if paymentIntentID.Valid {
		o.PaymentIntentID = &paymentIntentID.String
	}
	_ = json.Unmarshal(billingRaw, &o.BillingDetails)
	_ = json.Unmarshal(metaRaw, &o.Metadata)

	items, err := getOrderItems(ctx, db, orderID)
	if err != nil {
		return nil, err
	}
	o.Items = items
	return o, nil
}

// ListOrders returns all orders for a user, most recent first.
func ListOrders(ctx context.Context, db *sql.DB, userID string, limit, offset int) ([]Order, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var total int
	_ = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM commerce_orders WHERE user_id = $1`, userID,
	).Scan(&total)

	rows, err := db.QueryContext(ctx, `
		SELECT id, checkout_session_id, cart_id, user_id, trip_plan_id, status,
		       total_amount, refunded_amount, currency, payment_intent_id,
		       billing_details, metadata, created_at, updated_at
		FROM commerce_orders WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("ListOrders: %w", err)
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		var sessionID, tripPlanID, paymentIntentID sql.NullString
		var billingRaw, metaRaw []byte
		if err = rows.Scan(
			&o.ID, &sessionID, &o.CartID, &o.UserID, &tripPlanID, &o.Status,
			&o.TotalAmount, &o.RefundedAmount, &o.Currency, &paymentIntentID,
			&billingRaw, &metaRaw, &o.CreatedAt, &o.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		if sessionID.Valid {
			o.CheckoutSessionID = &sessionID.String
		}
		if tripPlanID.Valid {
			o.TripPlanID = &tripPlanID.String
		}
		if paymentIntentID.Valid {
			o.PaymentIntentID = &paymentIntentID.String
		}
		_ = json.Unmarshal(billingRaw, &o.BillingDetails)
		_ = json.Unmarshal(metaRaw, &o.Metadata)
		orders = append(orders, o)
	}
	if orders == nil {
		orders = []Order{}
	}
	return orders, total, nil
}

// CancelOrder sets the order status to 'cancelled' if currently pending or confirmed.
func CancelOrder(ctx context.Context, db *sql.DB, orderID, userID string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE commerce_orders
		SET status = 'cancelled'
		WHERE id = $1 AND user_id = $2
		  AND status IN ('pending', 'confirmed')
	`, orderID, userID)
	if err != nil {
		return fmt.Errorf("CancelOrder: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("order not found or cannot be cancelled")
	}
	return nil
}

// ─── private ──────────────────────────────────────────────────────────────────

func getOrderItems(ctx context.Context, db *sql.DB, orderID string) ([]OrderItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, order_id, item_type, item_id, tour_agent_booking_id, quantity,
		       unit_price, total_price, item_snapshot, status, failure_reason, traveler_date,
		       created_at::text
		FROM commerce_order_items WHERE order_id = $1 ORDER BY created_at
	`, orderID)
	if err != nil {
		return nil, fmt.Errorf("getOrderItems: %w", err)
	}
	defer rows.Close()

	var items []OrderItem
	for rows.Next() {
		var item OrderItem
		var bookingID, failureReason, travelerDate sql.NullString
		var snapshotRaw []byte
		if err = rows.Scan(
			&item.ID, &item.OrderID, &item.ItemType, &item.ItemID,
			&bookingID, &item.Quantity, &item.UnitPrice, &item.TotalPrice,
			&snapshotRaw, &item.Status, &failureReason, &travelerDate, &item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if bookingID.Valid {
			item.TourAgentBookingID = &bookingID.String
		}
		if failureReason.Valid {
			item.FailureReason = &failureReason.String
		}
		if travelerDate.Valid {
			item.TravelerDate = &travelerDate.String
		}
		_ = json.Unmarshal(snapshotRaw, &item.ItemSnapshot)
		items = append(items, item)
	}
	if items == nil {
		items = []OrderItem{}
	}
	return items, nil
}
