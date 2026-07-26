package checkout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// ─── tour-planner-ai client ───────────────────────────────────────────────────

// ValidateWithAI calls tour-planner-ai to check budget limits and scheduling
// conflicts for the items in the cart. It uses the budget_check intent so the
// BudgetAssistant handles the routing.
func ValidateWithAI(ctx context.Context, cartItems []map[string]any, tripContext map[string]any) (*AIValidationResult, error) {
	baseURL := os.Getenv("TOUR_PLANNER_AI_URL")
	if baseURL == "" {
		// Validation is advisory — return a permissive default when the service is unconfigured.
		return &AIValidationResult{Valid: true, BudgetStatus: "no_budget_set", AgentIntent: "budget"}, nil
	}

	payload := map[string]any{
		"prompt": "Validate checkout: check budget limits and scheduling conflicts for these cart items.",
		"context": mergeMaps(map[string]any{
			"track_budget": true,
			"cost_items":   cartItems,
		}, tripContext),
	}

	result, err := postJSON(ctx, baseURL+"/invoke", payload)
	if err != nil {
		// Validation failures are non-blocking so checkout can still proceed.
		return &AIValidationResult{Valid: true, BudgetStatus: "no_budget_set", AgentIntent: "budget"}, nil
	}

	validation := &AIValidationResult{AgentIntent: "budget", Valid: true, BudgetStatus: "no_budget_set"}
	if intent, ok := result["intent"].(string); ok {
		validation.AgentIntent = intent
	}

	if agentResp, ok := result["agent_response"].(map[string]any); ok {
		if status, ok := agentResp["budget_status"].(string); ok {
			validation.BudgetStatus = status
			if status == "over_budget" {
				validation.Valid = false
			}
		}
		if conflicts, ok := agentResp["conflicts"].([]any); ok && len(conflicts) > 0 {
			validation.ConflictsFound = true
			for _, c := range conflicts {
				if s, ok := c.(string); ok {
					validation.Conflicts = append(validation.Conflicts, s)
				}
			}
		}
		if warnings, ok := agentResp["warnings"].([]any); ok {
			for _, w := range warnings {
				if s, ok := w.(string); ok {
					validation.Warnings = append(validation.Warnings, s)
				}
			}
		}
	}
	return validation, nil
}

// ─── tour-agent client ────────────────────────────────────────────────────────

// CreateTourAgentBooking posts to tour-agent's booking endpoint for a single
// cart item. Returns the booking ID on success.
func CreateTourAgentBooking(ctx context.Context, req TourAgentBookingRequest) (*TourAgentBookingResponse, error) {
	baseURL := os.Getenv("TOUR_AGENT_API_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("TOUR_AGENT_API_URL not configured")
	}

	resp, err := postJSON(ctx, baseURL+"/bookings", req)
	if err != nil {
		return nil, fmt.Errorf("CreateTourAgentBooking: %w", err)
	}

	result := &TourAgentBookingResponse{}
	if id, ok := resp["booking_id"].(string); ok {
		result.BookingID = id
	}
	if status, ok := resp["status"].(string); ok {
		result.Status = status
	}
	return result, nil
}

// ─── shared HTTP helper ───────────────────────────────────────────────────────

func postJSON(ctx context.Context, url string, body any) (map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
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
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, string(respBytes))
	}

	var out map[string]any
	if err = json.Unmarshal(respBytes, &out); err != nil {
		return nil, err
	}
	return out, nil
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
