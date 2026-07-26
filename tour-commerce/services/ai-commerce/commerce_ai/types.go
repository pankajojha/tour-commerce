package commerce_ai

// RecommendRequest is the body for POST /commerce/ai/recommendations.
type RecommendRequest struct {
	CartID      string         `json:"cart_id"`
	Destination string         `json:"destination"`
	TripContext map[string]any `json:"trip_context,omitempty"`
	Interests   []string       `json:"interests,omitempty"`
	Budget      float64        `json:"budget,omitempty"`
}

// UpsellRequest is the body for POST /commerce/ai/upsell.
type UpsellRequest struct {
	CartID    string         `json:"cart_id"`
	ItemType  string         `json:"item_type"`
	ItemID    string         `json:"item_id"`
	TripCtx   map[string]any `json:"trip_context,omitempty"`
}

// BudgetCheckRequest is the body for POST /commerce/ai/budget-check.
type BudgetCheckRequest struct {
	CartID      string           `json:"cart_id"`
	BudgetLimit float64          `json:"budget_limit"`
	CostItems   []map[string]any `json:"cost_items"`
	TripContext map[string]any   `json:"trip_context,omitempty"`
}

// AIResponse is the normalised response structure returned to the client.
type AIResponse struct {
	Intent          string   `json:"intent"`
	Summary         string   `json:"summary,omitempty"`
	Recommendations []any    `json:"recommendations,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	BudgetStatus    string   `json:"budget_status,omitempty"`
	ConflictsFound  bool     `json:"conflicts_found,omitempty"`
	Conflicts       []string `json:"conflicts,omitempty"`
	RawResponse     any      `json:"raw_response,omitempty"`
}
