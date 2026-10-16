package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// SQL has been migrated to db-proxy (internal/queries/payments_query.go). This
// package now speaks to db-proxy's /api/payments/* domain APIs over HTTP; it
// sends parameters as JSON and never builds SQL.

// Client calls db-proxy domain APIs. Modeled on trips/handler/dbproxy_client.go:
// reads DB_PROXY_URL + DB_PROXY_API_KEY and POSTs JSON with a Bearer token.
type Client struct {
	baseURL      string
	apiKey       string
	clientID     string
	clientSecret string
	http         *http.Client
}

// ErrNotFound mirrors a 404 from the db-proxy domain API.
var ErrNotFound = errors.New("not found")

// Open returns a configured db-proxy Client.
func Open() (*Client, error) {
	base := normalizeProxyBaseURL(firstNonEmpty(
		os.Getenv("DB_PROXY_URL"), os.Getenv("DATABASE_PROXY_URL"),
		os.Getenv("DB_PROXY_ADDR"), os.Getenv("DATABASE_PROXY_ADDR"),
		proxyTarget(),
	))
	if base == "" {
		return nil, fmt.Errorf("db-proxy base URL not configured (set DB_PROXY_URL)")
	}
	return &Client{
		baseURL:      strings.TrimRight(base, "/"),
		apiKey:       os.Getenv("DB_PROXY_API_KEY"),
		clientID:     getenvDefault("DB_PROXY_CLIENT_ID", "johnpankaj"),
		clientSecret: getenvDefault("DB_PROXY_CLIENT_SECRET", "exploreragents2findmytour"),
		http:         &http.Client{Timeout: 15 * time.Second},
	}, nil
}

func proxyTarget() string {
	if host := firstNonEmpty(os.Getenv("DB_PROXY_HOST"), os.Getenv("DATABASE_PROXY_HOST")); host != "" {
		port := firstNonEmpty(os.Getenv("DB_PROXY_PORT"), os.Getenv("DATABASE_PROXY_PORT"), "8090")
		return host + ":" + port
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// getenvDefault returns the env var value or fallback when unset/empty.
func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func normalizeProxyBaseURL(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "http://") || strings.HasPrefix(v, "https://") {
		return v
	}
	if strings.HasSuffix(v, ":443") || strings.EqualFold(os.Getenv("DB_PROXY_TLS"), "true") {
		return "https://" + v
	}
	return "http://" + v
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	if c.clientID != "" {
		req.Header.Set("X-Client-Id", c.clientID)
	}
	if c.clientSecret != "" {
		req.Header.Set("X-Client-Secret", c.clientSecret)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		if out == nil {
			return nil
		}
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode %s response: %w", path, err)
		}
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("db-proxy %s: status %d: %s", path, resp.StatusCode, strings.TrimSpace(string(data)))
	}
}

// InsertPayment records a new payment intent in commerce_payments.
func InsertPayment(ctx context.Context, c *Client, orderID, providerPaymentID, currency string, amount float64) (string, error) {
	var resp struct {
		ID string `json:"id"`
	}
	err := c.post(ctx, "/api/payments/insert", map[string]any{
		"order_id":            orderID,
		"provider_payment_id": providerPaymentID,
		"currency":            currency,
		"amount":              amount,
	}, &resp)
	if err != nil {
		return "", fmt.Errorf("InsertPayment: %w", err)
	}
	return resp.ID, nil
}

// UpdatePaymentStatus updates status and optional charge/card details.
func UpdatePaymentStatus(ctx context.Context, c *Client, providerPaymentID, status, chargeID, pmType, last4 string) error {
	if err := c.post(ctx, "/api/payments/update-status", map[string]string{
		"provider_payment_id": providerPaymentID,
		"status":              status,
		"charge_id":           chargeID,
		"pm_type":             pmType,
		"last4":               last4,
	}, nil); err != nil {
		return fmt.Errorf("UpdatePaymentStatus: %w", err)
	}
	return nil
}

// UpdateOrderStatus sets status on the parent order after payment confirmation.
func UpdateOrderStatus(ctx context.Context, c *Client, orderID, status string) error {
	if err := c.post(ctx, "/api/payments/update-order-status", map[string]string{
		"order_id": orderID,
		"status":   status,
	}, nil); err != nil {
		return fmt.Errorf("UpdateOrderStatus: %w", err)
	}
	return nil
}

// UpdateRefundedAmount increments the refunded_amount column on the payment row.
func UpdateRefundedAmount(ctx context.Context, c *Client, providerPaymentID string, refundAmt float64) error {
	if err := c.post(ctx, "/api/payments/update-refunded-amount", map[string]any{
		"provider_payment_id": providerPaymentID,
		"refund_amount":       refundAmt,
	}, nil); err != nil {
		return fmt.Errorf("UpdateRefundedAmount: %w", err)
	}
	return nil
}

// GetPaymentByProviderID returns the payment record for an idempotency check.
// Returns (nil, nil) when absent, preserving the original contract.
func GetPaymentByProviderID(ctx context.Context, c *Client, providerPaymentID string) (*PaymentRecord, error) {
	var p PaymentRecord
	err := c.post(ctx, "/api/payments/get-by-provider-id", map[string]string{
		"provider_payment_id": providerPaymentID,
	}, &p)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetPaymentByProviderID: %w", err)
	}
	return &p, nil
}

// GetSucceededPaymentIntentForOrder returns the Stripe PaymentIntent ID for a
// succeeded payment on the given order. Returns "" when none is found.
func GetSucceededPaymentIntentForOrder(ctx context.Context, c *Client, orderID string) (string, error) {
	var resp struct {
		ProviderPaymentID string `json:"provider_payment_id"`
	}
	err := c.post(ctx, "/api/payments/succeeded-intent-for-order", map[string]string{
		"order_id": orderID,
	}, &resp)
	if errors.Is(err, ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("GetSucceededPaymentIntentForOrder: %w", err)
	}
	return resp.ProviderPaymentID, nil
}
