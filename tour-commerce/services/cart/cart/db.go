package cart

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

// SQL has been migrated to db-proxy (internal/queries/cart_query.go). This
// package now speaks to db-proxy's /api/cart/* domain APIs over HTTP; it sends
// parameters as JSON and never builds SQL.

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

// Open returns a configured db-proxy Client. Kept as Open() so callers do not
// change their initialization flow.
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

// post sends a JSON body to the given /api path and decodes the JSON response
// into out. A 404 is surfaced as ErrNotFound so callers can preserve their
// "not found" response shapes.
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

// ─── domain calls (same signatures as before, minus the SQL) ──────────────────

// GetCart fetches a cart with its items by cart ID. Returns (nil, nil) when the
// cart is absent, preserving the original contract.
func GetCart(ctx context.Context, c *Client, cartID string) (*Cart, error) {
	var cart Cart
	err := c.post(ctx, "/api/cart/get", map[string]string{"cart_id": cartID}, &cart)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetCart: %w", err)
	}
	return &cart, nil
}

// GetActiveCartForUser returns the user's active cart, or nil.
func GetActiveCartForUser(ctx context.Context, c *Client, userID string) (*Cart, error) {
	var cart Cart
	err := c.post(ctx, "/api/cart/active", map[string]string{"user_id": userID}, &cart)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetActiveCartForUser: %w", err)
	}
	return &cart, nil
}

// CreateCart inserts a new active cart and returns it.
func CreateCart(ctx context.Context, c *Client, userID string, req CreateCartRequest) (*Cart, error) {
	tripPlanID := ""
	if req.TripPlanID != nil {
		tripPlanID = *req.TripPlanID
	}
	var cart Cart
	err := c.post(ctx, "/api/cart/create", map[string]any{
		"user_id":      userID,
		"trip_plan_id": tripPlanID,
		"currency":     req.Currency,
	}, &cart)
	if err != nil {
		return nil, fmt.Errorf("CreateCart: %w", err)
	}
	return &cart, nil
}

// AddItem inserts a cart item and returns it.
func AddItem(ctx context.Context, c *Client, cartID string, req AddItemRequest) (*CartItem, error) {
	travelerDate := ""
	if req.TravelerDate != nil {
		travelerDate = *req.TravelerDate
	}
	var item CartItem
	err := c.post(ctx, "/api/cart/add-item", map[string]any{
		"cart_id":                   cartID,
		"item_type":                 req.ItemType,
		"item_id":                   req.ItemID,
		"quantity":                  req.Quantity,
		"unit_price":                req.UnitPrice,
		"item_snapshot":             req.ItemSnapshot,
		"ai_recommended":            req.AIRecommended,
		"ai_recommendation_context": req.AIRecommendationContext,
		"traveler_date":             travelerDate,
	}, &item)
	if err != nil {
		return nil, fmt.Errorf("AddItem: %w", err)
	}
	return &item, nil
}

// RemoveItem deletes a cart item by ID.
func RemoveItem(ctx context.Context, c *Client, cartID, itemID string) error {
	err := c.post(ctx, "/api/cart/remove-item", map[string]string{
		"cart_id": cartID,
		"item_id": itemID,
	}, nil)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("item not found")
	}
	if err != nil {
		return fmt.Errorf("RemoveItem: %w", err)
	}
	return nil
}

// UpdateItemQuantity changes quantity and recomputes total_price.
func UpdateItemQuantity(ctx context.Context, c *Client, cartID, itemID string, quantity int) (*CartItem, error) {
	var item CartItem
	err := c.post(ctx, "/api/cart/update-item-quantity", map[string]any{
		"cart_id":  cartID,
		"item_id":  itemID,
		"quantity": quantity,
	}, &item)
	if errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("item not found")
	}
	if err != nil {
		return nil, fmt.Errorf("UpdateItemQuantity: %w", err)
	}
	return &item, nil
}

// MarkCartCheckedOut sets status = 'checked_out'.
func MarkCartCheckedOut(ctx context.Context, c *Client, cartID string) error {
	if err := c.post(ctx, "/api/cart/mark-checked-out", map[string]string{"cart_id": cartID}, nil); err != nil {
		return fmt.Errorf("MarkCartCheckedOut: %w", err)
	}
	return nil
}
