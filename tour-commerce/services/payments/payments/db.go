package payments

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Open returns a connected *sql.DB.
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

// InsertPayment records a new payment intent in commerce_payments.
func InsertPayment(ctx context.Context, db *sql.DB, orderID, providerPaymentID, currency string, amount float64) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `
		INSERT INTO commerce_payments (order_id, provider_payment_id, amount, currency, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id
	`, orderID, providerPaymentID, amount, currency).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("InsertPayment: %w", err)
	}
	return id, nil
}

// UpdatePaymentStatus updates status and optional charge/card details after a webhook event.
func UpdatePaymentStatus(ctx context.Context, db *sql.DB, providerPaymentID, status, chargeID, pmType, last4 string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE commerce_payments
		SET status              = $2,
		    provider_charge_id  = COALESCE(NULLIF($3,''), provider_charge_id),
		    payment_method_type = COALESCE(NULLIF($4,''), payment_method_type),
		    last4               = COALESCE(NULLIF($5,''), last4)
		WHERE provider_payment_id = $1
	`, providerPaymentID, status, chargeID, pmType, last4)
	return err
}

// UpdateOrderStatus sets status on the parent order after payment confirmation.
func UpdateOrderStatus(ctx context.Context, db *sql.DB, orderID, status string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE commerce_orders SET status = $2 WHERE id = $1`, orderID, status)
	return err
}

// UpdateRefundedAmount increments the refunded_amount column on the payment row.
func UpdateRefundedAmount(ctx context.Context, db *sql.DB, providerPaymentID string, refundAmt float64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE commerce_payments
		SET refunded_amount = refunded_amount + $2,
		    status = CASE
		        WHEN refunded_amount + $2 >= amount THEN 'refunded'
		        ELSE 'partially_refunded'
		    END
		WHERE provider_payment_id = $1
	`, providerPaymentID, refundAmt)
	return err
}

// GetPaymentByProviderID returns the payment record for an idempotency check.
func GetPaymentByProviderID(ctx context.Context, db *sql.DB, providerPaymentID string) (*PaymentRecord, error) {
	p := &PaymentRecord{}
	var chargeID, pmType, last4 sql.NullString
	err := db.QueryRowContext(ctx, `
		SELECT id, order_id, provider, provider_payment_id, provider_charge_id,
		       amount, refunded_amount, currency, status, payment_method_type, last4, created_at::text
		FROM commerce_payments WHERE provider_payment_id = $1 AND provider = 'stripe'
	`, providerPaymentID).Scan(
		&p.ID, &p.OrderID, &p.Provider, &p.ProviderPaymentID, &chargeID,
		&p.Amount, &p.RefundedAmount, &p.Currency, &p.Status,
		&pmType, &last4, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetPaymentByProviderID: %w", err)
	}
	if chargeID.Valid {
		p.ProviderChargeID = &chargeID.String
	}
	if pmType.Valid {
		p.PaymentMethodType = &pmType.String
	}
	if last4.Valid {
		p.Last4 = &last4.String
	}
	return p, nil
}
