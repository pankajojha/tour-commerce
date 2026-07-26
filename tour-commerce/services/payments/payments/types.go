package payments

// CreateIntentRequest is the body for POST /commerce/payments/intent.
type CreateIntentRequest struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// CreateIntentResponse is returned after a Stripe PaymentIntent is created.
type CreateIntentResponse struct {
	PaymentIntentID string  `json:"payment_intent_id"`
	ClientSecret    string  `json:"client_secret"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Status          string  `json:"status"`
}

// RefundRequest is the body for POST /commerce/payments/{order_id}/refund.
type RefundRequest struct {
	Amount float64 `json:"amount,omitempty"` // partial refund; 0 = full refund
	Reason string  `json:"reason,omitempty"`
}

// PaymentRecord mirrors the commerce_payments table.
type PaymentRecord struct {
	ID                string  `json:"id"`
	OrderID           string  `json:"order_id"`
	Provider          string  `json:"provider"`
	ProviderPaymentID string  `json:"provider_payment_id"`
	ProviderChargeID  *string `json:"provider_charge_id,omitempty"`
	Amount            float64 `json:"amount"`
	RefundedAmount    float64 `json:"refunded_amount"`
	Currency          string  `json:"currency"`
	Status            string  `json:"status"`
	PaymentMethodType *string `json:"payment_method_type,omitempty"`
	Last4             *string `json:"last4,omitempty"`
	CreatedAt         string  `json:"created_at"`
}
