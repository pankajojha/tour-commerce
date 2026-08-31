package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/paymentintent"
	"github.com/stripe/stripe-go/v74/refund"
	"github.com/stripe/stripe-go/v74/webhook"

	paypkg "github.com/explorer-agent/commerce-payments/payments"
)

var db *sql.DB

func init() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	var err error
	db, err = paypkg.Open()
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
}

type routeHandler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	routes := map[string]routeHandler{
		"POST /commerce/payments/intent":            handleCreateIntent,
		"POST /commerce/payments/webhook":           handleWebhook,
		"POST /commerce/payments/{order_id}/refund": handleRefund,
	}
	key := matchRoute(req.HTTPMethod, req.Path)
	if h, ok := routes[key]; ok {
		return h(ctx, req)
	}
	return jsonError(http.StatusNotFound, "route not found"), nil
}

func matchRoute(method, path string) string {
	m := strings.ToUpper(method)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case m == "POST" && len(parts) == 3 && parts[2] == "intent":
		return "POST /commerce/payments/intent"
	case m == "POST" && len(parts) == 3 && parts[2] == "webhook":
		return "POST /commerce/payments/webhook"
	case m == "POST" && len(parts) == 4 && parts[3] == "refund":
		return "POST /commerce/payments/{order_id}/refund"
	}
	return m + " " + path
}

// ─────────────────────────────────────────────
// POST /commerce/payments/intent
// Creates a Stripe PaymentIntent for the given order amount.
// ─────────────────────────────────────────────

func handleCreateIntent(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	_, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body paypkg.CreateIntentRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil || body.OrderID == "" {
		return jsonError(http.StatusBadRequest, "order_id and amount are required"), nil
	}
	if body.Amount <= 0 {
		return jsonError(http.StatusBadRequest, "amount must be positive"), nil
	}
	cur := strings.ToLower(body.Currency)
	if cur == "" {
		cur = "usd"
	}

	// Stripe amounts are in the smallest currency unit (cents for USD).
	amountCents := int64(body.Amount * 100)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountCents),
		Currency: stripe.String(cur),
		Params: stripe.Params{
			Metadata: map[string]string{"order_id": body.OrderID},
		},
	}
	pi, stripeErr := paymentintent.New(params)
	if stripeErr != nil {
		log.Printf("Stripe PaymentIntent create error: %v", stripeErr)
		return jsonError(http.StatusBadGateway, "payment provider error"), nil
	}

	_, dbErr := paypkg.InsertPayment(ctx, db, body.OrderID, pi.ID, cur, body.Amount)
	if dbErr != nil {
		log.Printf("InsertPayment: %v", dbErr)
	}

	return jsonOK(paypkg.CreateIntentResponse{
		PaymentIntentID: pi.ID,
		ClientSecret:    pi.ClientSecret,
		Amount:          body.Amount,
		Currency:        cur,
		Status:          string(pi.Status),
	})
}

// ─────────────────────────────────────────────
// POST /commerce/payments/webhook
// Stripe sends events here; we verify the signature and update DB state.
// ─────────────────────────────────────────────

func handleWebhook(_ context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	sig := req.Headers["Stripe-Signature"]

	event, err := webhook.ConstructEvent([]byte(req.Body), sig, webhookSecret)
	if err != nil {
		log.Printf("Stripe webhook signature verification failed: %v", err)
		return jsonError(http.StatusBadRequest, "invalid webhook signature"), nil
	}

	ctx := context.Background()

	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err = json.Unmarshal(event.Data.Raw, &pi); err == nil {
			chargeID := ""
			pmType := ""
			last4 := ""
			if pi.LatestCharge != nil {
				chargeID = pi.LatestCharge.ID
				if pi.LatestCharge.PaymentMethodDetails != nil && pi.LatestCharge.PaymentMethodDetails.Card != nil {
					pmType = "card"
					last4 = pi.LatestCharge.PaymentMethodDetails.Card.Last4
				}
			}
			_ = paypkg.UpdatePaymentStatus(ctx, db, pi.ID, "succeeded", chargeID, pmType, last4)
			if orderID, ok := pi.Metadata["order_id"]; ok {
				_ = paypkg.UpdateOrderStatus(ctx, db, orderID, "confirmed")
			}
		}

	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err = json.Unmarshal(event.Data.Raw, &pi); err == nil {
			_ = paypkg.UpdatePaymentStatus(ctx, db, pi.ID, "failed", "", "", "")
			if orderID, ok := pi.Metadata["order_id"]; ok {
				_ = paypkg.UpdateOrderStatus(ctx, db, orderID, "failed")
			}
		}

	case "charge.refunded":
		var ch stripe.Charge
		if err = json.Unmarshal(event.Data.Raw, &ch); err == nil && ch.PaymentIntent != nil {
			refundAmt := float64(ch.AmountRefunded) / 100
			_ = paypkg.UpdateRefundedAmount(ctx, db, ch.PaymentIntent.ID, refundAmt)
		}
	}

	return events.APIGatewayProxyResponse{StatusCode: http.StatusOK, Body: `{"received":true}`}, nil
}

// ─────────────────────────────────────────────
// POST /commerce/payments/{order_id}/refund
// ─────────────────────────────────────────────

func handleRefund(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	orderID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(orderID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid order id"), nil
	}

	_, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body paypkg.RefundRequest
	if req.Body != "" {
		_ = json.Unmarshal([]byte(req.Body), &body)
	}

	// Fetch the payment to get the Stripe PaymentIntent ID.
	var paymentIntentID string
	_ = db.QueryRowContext(ctx,
		`SELECT provider_payment_id FROM commerce_payments WHERE order_id = $1 AND status = 'succeeded'`,
		orderID,
	).Scan(&paymentIntentID)
	if paymentIntentID == "" {
		return jsonError(http.StatusNotFound, "no succeeded payment found for this order"), nil
	}

	params := &stripe.RefundParams{PaymentIntent: stripe.String(paymentIntentID)}
	if body.Amount > 0 {
		params.Amount = stripe.Int64(int64(body.Amount * 100))
	}

	rf, stripeErr := refund.New(params)
	if stripeErr != nil {
		log.Printf("Stripe refund error: %v", stripeErr)
		return jsonError(http.StatusBadGateway, "payment provider error"), nil
	}

	return jsonOK(map[string]any{
		"refund_id": rf.ID,
		"amount":    float64(rf.Amount) / 100,
		"status":    string(rf.Status),
		"order_id":  orderID,
	})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func pathSegment(path string, idx int) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if idx < len(parts) {
		return parts[idx]
	}
	return ""
}

func subFromRequest(req events.APIGatewayProxyRequest) (uuid.UUID, error) {
	sub, ok := req.RequestContext.Authorizer["sub"].(string)
	if !ok || sub == "" {
		return uuid.Nil, fmt.Errorf("sub claim missing")
	}
	return uuid.Parse(sub)
}

// ioReadAll reads the body of an http.Response and discards the rest.
// Included so the import of io is valid across Go versions.
var _ = io.Discard

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func jsonOK(v any) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       mustJSON(v),
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
