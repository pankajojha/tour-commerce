package checkout

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

// SQL has been migrated to db-proxy (internal/queries/checkout_query.go). This
// package now speaks to db-proxy's /api/checkout/* domain APIs over HTTP; it
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

// CartSummary is the minimal cart info needed for checkout.
type CartSummary struct {
	ID          string            `json:"id"`
	UserID      string            `json:"user_id"`
	TripPlanID  string            `json:"trip_plan_id"`
	Status      string            `json:"status"`
	Currency    string            `json:"currency"`
	TotalAmount float64           `json:"total_amount"`
	Items       []CartItemSummary `json:"items"`
}

// CartItemSummary is the minimal item info needed to create bookings.
type CartItemSummary struct {
	ID           string         `json:"id"`
	ItemType     string         `json:"item_type"`
	ItemID       string         `json:"item_id"`
	Quantity     int            `json:"quantity"`
	UnitPrice    float64        `json:"unit_price"`
	TotalPrice   float64        `json:"total_price"`
	ItemSnapshot map[string]any `json:"item_snapshot"`
	TravelerDate string         `json:"traveler_date"`
}

// GetCartSummary loads the cart and all its items for checkout validation.
// Returns (nil, nil) when the cart is absent, preserving the original contract.
func GetCartSummary(ctx context.Context, c *Client, cartID, userID string) (*CartSummary, error) {
	var cs CartSummary
	err := c.post(ctx, "/api/checkout/cart-summary", map[string]string{
		"cart_id": cartID,
		"user_id": userID,
	}, &cs)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetCartSummary: %w", err)
	}
	return &cs, nil
}

// CreateSession inserts a checkout session and returns it.
func CreateSession(ctx context.Context, c *Client, cartID, userID string, totalAmount float64, currency string, billing any) (*Session, error) {
	var s Session
	err := c.post(ctx, "/api/checkout/create-session", map[string]any{
		"cart_id":         cartID,
		"user_id":         userID,
		"total_amount":    totalAmount,
		"currency":        currency,
		"billing_details": billing,
	}, &s)
	if err != nil {
		return nil, fmt.Errorf("CreateSession: %w", err)
	}
	return &s, nil
}

// GetSession retrieves a checkout session by ID. Returns (nil, nil) when absent.
func GetSession(ctx context.Context, c *Client, sessionID string) (*Session, error) {
	var s Session
	err := c.post(ctx, "/api/checkout/get-session", map[string]string{"session_id": sessionID}, &s)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetSession: %w", err)
	}
	return &s, nil
}

// UpdateSessionStatus sets the session status and optionally stores AI validation.
func UpdateSessionStatus(ctx context.Context, c *Client, sessionID string, status SessionStatus, aiValidation any) error {
	if err := c.post(ctx, "/api/checkout/update-session-status", map[string]any{
		"session_id":    sessionID,
		"status":        string(status),
		"ai_validation": aiValidation,
	}, nil); err != nil {
		return fmt.Errorf("UpdateSessionStatus: %w", err)
	}
	return nil
}

// orderItemInput mirrors db-proxy's CheckoutOrderItemInput JSON contract.
type orderItemInput struct {
	ItemType     string `json:"item_type"`
	ItemID       string `json:"item_id"`
	Quantity     int    `json:"quantity"`
	UnitPrice    float64 `json:"unit_price"`
	TotalPrice   float64 `json:"total_price"`
	ItemSnapshot any    `json:"item_snapshot"`
	TravelerDate string `json:"traveler_date"`
}

// CreateOrder inserts an order (and its items) from a confirmed checkout session.
func CreateOrder(ctx context.Context, c *Client, session *Session, cart *CartSummary, paymentIntentID string) (string, error) {
	items := make([]orderItemInput, 0, len(cart.Items))
	for _, item := range cart.Items {
		items = append(items, orderItemInput{
			ItemType:     item.ItemType,
			ItemID:       item.ItemID,
			Quantity:     item.Quantity,
			UnitPrice:    item.UnitPrice,
			TotalPrice:   item.TotalPrice,
			ItemSnapshot: item.ItemSnapshot,
			TravelerDate: item.TravelerDate,
		})
	}
	var resp struct {
		OrderID string `json:"order_id"`
	}
	err := c.post(ctx, "/api/checkout/create-order", map[string]any{
		"session_id":        session.ID,
		"cart_id":           session.CartID,
		"user_id":           session.UserID,
		"trip_plan_id":      cart.TripPlanID,
		"total_amount":      session.TotalAmount,
		"currency":          session.Currency,
		"payment_intent_id": paymentIntentID,
		"billing_details":   session.BillingDetails,
		"items":             items,
	}, &resp)
	if err != nil {
		return "", fmt.Errorf("CreateOrder: %w", err)
	}
	return resp.OrderID, nil
}

// ConfirmOrderItem updates a single order item with the tour-agent booking ID.
func ConfirmOrderItem(ctx context.Context, c *Client, orderID, itemID, tourAgentBookingID string) error {
	if err := c.post(ctx, "/api/checkout/confirm-order-item", map[string]string{
		"order_id":              orderID,
		"item_id":               itemID,
		"tour_agent_booking_id": tourAgentBookingID,
	}, nil); err != nil {
		return fmt.Errorf("ConfirmOrderItem: %w", err)
	}
	return nil
}

// SetOrderStatus updates the top-level order status.
func SetOrderStatus(ctx context.Context, c *Client, orderID, status string) error {
	if err := c.post(ctx, "/api/checkout/set-order-status", map[string]string{
		"order_id": orderID,
		"status":   status,
	}, nil); err != nil {
		return fmt.Errorf("SetOrderStatus: %w", err)
	}
	return nil
}

// LogAIInteraction persists an AI commerce interaction for audit.
func LogAIInteraction(ctx context.Context, c *Client, userID, cartID, interactionType string, request, response any, agentIntent string) error {
	if err := c.post(ctx, "/api/checkout/log-ai-interaction", map[string]any{
		"user_id":          userID,
		"cart_id":          cartID,
		"interaction_type": interactionType,
		"request":          request,
		"response":         response,
		"agent_intent":     agentIntent,
	}, nil); err != nil {
		return fmt.Errorf("LogAIInteraction: %w", err)
	}
	return nil
}
