package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	ai "github.com/explorer-agent/commerce-ai/commerce_ai"
)

var db *sql.DB

func init() {
	var err error
	db, err = ai.Open()
	if err != nil {
		log.Printf("DB open warning (audit logging disabled): %v", err)
	}
}

type routeHandler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	routes := map[string]routeHandler{
		"POST /commerce/ai/recommendations": handleRecommendations,
		"POST /commerce/ai/budget-check":    handleBudgetCheck,
		"POST /commerce/ai/upsell":          handleUpsell,
	}
	key := matchRoute(req.HTTPMethod, req.Path)
	if h, ok := routes[key]; ok {
		return h(ctx, req)
	}
	return jsonError(http.StatusNotFound, "route not found"), nil
}

func matchRoute(method, path string) string {
	m := strings.ToUpper(method)
	switch path {
	case "/commerce/ai/recommendations":
		return m + " /commerce/ai/recommendations"
	case "/commerce/ai/budget-check":
		return m + " /commerce/ai/budget-check"
	case "/commerce/ai/upsell":
		return m + " /commerce/ai/upsell"
	}
	return m + " " + path
}

// ─────────────────────────────────────────────
// POST /commerce/ai/recommendations
// Calls tour-planner-ai ExcursionAssistant to suggest items for the cart.
// ─────────────────────────────────────────────

func handleRecommendations(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body ai.RecommendRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.Destination == "" {
		return jsonError(http.StatusBadRequest, "destination is required"), nil
	}

	result, aiErr := ai.Recommend(ctx, body)
	if aiErr != nil {
		log.Printf("Recommend: %v", aiErr)
		return jsonError(http.StatusBadGateway, "AI service error"), nil
	}

	if db != nil {
		ai.LogInteraction(ctx, db, userID.String(), body.CartID, "recommendation", body, result, result.Intent)
	}
	return jsonOK(result)
}

// ─────────────────────────────────────────────
// POST /commerce/ai/budget-check
// Calls tour-planner-ai BudgetAssistant to validate cart total vs limit.
// ─────────────────────────────────────────────

func handleBudgetCheck(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body ai.BudgetCheckRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.BudgetLimit <= 0 {
		return jsonError(http.StatusBadRequest, "budget_limit must be positive"), nil
	}

	result, aiErr := ai.CheckBudget(ctx, body)
	if aiErr != nil {
		log.Printf("CheckBudget: %v", aiErr)
		return jsonError(http.StatusBadGateway, "AI service error"), nil
	}

	if db != nil {
		ai.LogInteraction(ctx, db, userID.String(), body.CartID, "budget_check", body, result, result.Intent)
	}
	return jsonOK(result)
}

// ─────────────────────────────────────────────
// POST /commerce/ai/upsell
// Calls tour-planner-ai to suggest complementary items for a cart item.
// ─────────────────────────────────────────────

func handleUpsell(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body ai.UpsellRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil || body.ItemID == "" {
		return jsonError(http.StatusBadRequest, "item_type and item_id are required"), nil
	}

	result, aiErr := ai.Upsell(ctx, body)
	if aiErr != nil {
		log.Printf("Upsell: %v", aiErr)
		return jsonError(http.StatusBadGateway, "AI service error"), nil
	}

	if db != nil {
		ai.LogInteraction(ctx, db, userID.String(), body.CartID, "upsell", body, result, result.Intent)
	}
	return jsonOK(result)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func subFromRequest(req events.APIGatewayProxyRequest) (uuid.UUID, error) {
	sub, ok := req.RequestContext.Authorizer["sub"].(string)
	if !ok || sub == "" {
		return uuid.Nil, fmt.Errorf("sub claim missing")
	}
	return uuid.Parse(sub)
}

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
