package commerce_ai

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

// Open returns a connected *sql.DB for audit logging.
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

// LogInteraction persists an AI commerce interaction for audit.
func LogInteraction(ctx context.Context, db *sql.DB, userID, cartID, interactionType string, request, response any, agentIntent string) {
	reqRaw, _ := json.Marshal(request)
	respRaw, _ := json.Marshal(response)
	_, _ = db.ExecContext(ctx, `
		INSERT INTO commerce_ai_interactions
		    (user_id, cart_id, interaction_type, request_payload, response_payload, agent_intent)
		VALUES ($1,$2,$3,$4,$5,$6)
	`, userID, nullStr(cartID), interactionType, reqRaw, respRaw, nullStr(agentIntent))
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

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
