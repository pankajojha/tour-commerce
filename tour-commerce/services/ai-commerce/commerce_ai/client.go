package commerce_ai

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

// The audit-logging SQL has been migrated to db-proxy
// (internal/queries/ai_query.go). This package now speaks to db-proxy's
// /api/ai/* domain API over HTTP; it sends parameters as JSON and never builds
// SQL. The tour-planner-ai calls below remain plain upstream HTTP requests.

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

// Open returns a configured db-proxy Client for audit logging. Kept as Open()
// so callers do not change their initialization flow.
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

// Recommend calls tour-planner-ai with an excursion/activity recommendation prompt
// shaped for the commerce context.
func Recommend(ctx context.Context, req RecommendRequest) (*AIResponse, error) {
	prompt := fmt.Sprintf(
		"Recommend shore excursions and activities for %s.",
		req.Destination,
	)
	if len(req.Interests) > 0 {
		prompt += fmt.Sprintf(" Interests: %v.", req.Interests)
	}
	if req.Budget > 0 {
		prompt += fmt.Sprintf(" Budget per person: $%.2f.", req.Budget)
	}

	aiCtx := mergeMaps(req.TripContext, map[string]any{
		"destination": req.Destination,
	})
	if req.Budget > 0 {
		aiCtx["budget"] = req.Budget
	}

	raw, err := invokeAI(ctx, prompt, aiCtx)
	if err != nil {
		return nil, err
	}
	return parseAIResponse(raw, "excursion"), nil
}

// CheckBudget calls tour-planner-ai's BudgetAssistant to assess whether the
// cart items fit within the given budget limit.
func CheckBudget(ctx context.Context, req BudgetCheckRequest) (*AIResponse, error) {
	prompt := fmt.Sprintf(
		"Check if the cart items fit within a total budget of $%.2f.",
		req.BudgetLimit,
	)
	aiCtx := mergeMaps(req.TripContext, map[string]any{
		"track_budget": true,
		"budget_limit": req.BudgetLimit,
		"cost_items":   req.CostItems,
	})

	raw, err := invokeAI(ctx, prompt, aiCtx)
	if err != nil {
		return nil, err
	}
	return parseAIResponse(raw, "budget"), nil
}

// Upsell calls tour-planner-ai to suggest complementary items for an existing
// cart item (e.g., a restaurant recommendation after booking an excursion).
func Upsell(ctx context.Context, req UpsellRequest) (*AIResponse, error) {
	prompt := fmt.Sprintf(
		"Suggest complementary activities or dining options to go with the %s booking %s.",
		req.ItemType, req.ItemID,
	)
	raw, err := invokeAI(ctx, prompt, req.TripCtx)
	if err != nil {
		return nil, err
	}
	return parseAIResponse(raw, "excursion"), nil
}

// LogInteraction persists an AI commerce interaction for audit via db-proxy.
// Best-effort: logging failures are swallowed so the AI flow is never blocked.
func LogInteraction(ctx context.Context, c *Client, userID, cartID, interactionType string, request, response any, agentIntent string) {
	if c == nil {
		return
	}
	body := map[string]any{
		"user_id":          userID,
		"cart_id":          cartID,
		"interaction_type": interactionType,
		"request":          request,
		"response":         response,
		"agent_intent":     agentIntent,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/ai/log-interaction", bytes.NewReader(raw))
	if err != nil {
		return
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
		return
	}
	_ = resp.Body.Close()
}

// ─── private helpers ──────────────────────────────────────────────────────────

func invokeAI(ctx context.Context, prompt string, aiCtx map[string]any) (map[string]any, error) {
	baseURL := os.Getenv("TOUR_PLANNER_AI_URL")
	if baseURL == "" {
		// Return a stub so that the commerce flow is never blocked when AI is unavailable.
		return map[string]any{
			"intent":         "excursion",
			"agent_response": map[string]any{"message": "AI service not configured"},
		}, nil
	}

	payload := map[string]any{
		"prompt":  prompt,
		"context": aiCtx,
	}
	raw, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/invoke", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tour-planner-ai returned %d: %s", resp.StatusCode, string(respBytes))
	}
	var out map[string]any
	_ = json.Unmarshal(respBytes, &out)
	return out, nil
}

func parseAIResponse(raw map[string]any, defaultIntent string) *AIResponse {
	result := &AIResponse{
		Intent:      defaultIntent,
		RawResponse: raw,
	}
	if intent, ok := raw["intent"].(string); ok {
		result.Intent = intent
	}
	if agentResp, ok := raw["agent_response"].(map[string]any); ok {
		if msg, ok := agentResp["message"].(string); ok {
			result.Summary = msg
		}
		if status, ok := agentResp["budget_status"].(string); ok {
			result.BudgetStatus = status
		}
		if conflicts, ok := agentResp["conflicts"].([]any); ok && len(conflicts) > 0 {
			result.ConflictsFound = true
			for _, c := range conflicts {
				if s, ok := c.(string); ok {
					result.Conflicts = append(result.Conflicts, s)
				}
			}
		}
		if warnings, ok := agentResp["warnings"].([]any); ok {
			for _, w := range warnings {
				if s, ok := w.(string); ok {
					result.Warnings = append(result.Warnings, s)
				}
			}
		}
		if recs, ok := agentResp["recommendations"].([]any); ok {
			result.Recommendations = recs
		}
	}
	return result
}

func mergeMaps(a, b map[string]any) map[string]any {
	out := make(map[string]any, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
