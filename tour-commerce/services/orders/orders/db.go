package orders

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

// SQL has been migrated to db-proxy (internal/queries/orders_query.go). This
// package now speaks to db-proxy's /api/orders/* domain APIs over HTTP; it
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

// GetOrder returns an order with all its items. Returns (nil, nil) when absent,
// preserving the original contract.
func GetOrder(ctx context.Context, c *Client, orderID string) (*Order, error) {
	var o Order
	err := c.post(ctx, "/api/orders/get", map[string]string{"order_id": orderID}, &o)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetOrder: %w", err)
	}
	return &o, nil
}

// ListOrders returns all orders for a user, most recent first, plus the total.
func ListOrders(ctx context.Context, c *Client, userID string, limit, offset int) ([]Order, int, error) {
	var resp struct {
		Orders []Order `json:"orders"`
		Total  int     `json:"total"`
	}
	err := c.post(ctx, "/api/orders/list", map[string]any{
		"user_id": userID,
		"limit":   limit,
		"offset":  offset,
	}, &resp)
	if err != nil {
		return nil, 0, fmt.Errorf("ListOrders: %w", err)
	}
	if resp.Orders == nil {
		resp.Orders = []Order{}
	}
	return resp.Orders, resp.Total, nil
}

// CancelOrder sets the order status to 'cancelled' if pending or confirmed.
func CancelOrder(ctx context.Context, c *Client, orderID, userID string) error {
	err := c.post(ctx, "/api/orders/cancel", map[string]string{
		"order_id": orderID,
		"user_id":  userID,
	}, nil)
	if errors.Is(err, ErrNotFound) {
		return fmt.Errorf("order not found or cannot be cancelled")
	}
	if err != nil {
		return fmt.Errorf("CancelOrder: %w", err)
	}
	return nil
}
