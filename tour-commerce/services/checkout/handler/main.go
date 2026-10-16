package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	co "github.com/explorer-agent/commerce-checkout/checkout"
)

var db *co.Client

func init() {
	var err error
	db, err = co.Open()
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
}

// ─────────────────────────────────────────────
// Router
// ─────────────────────────────────────────────

type routeHandler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	routes := map[string]routeHandler{
		"POST /commerce/checkout":                    handleInitiate,
		"GET /commerce/checkout/{session_id}":        handleGetSession,
		"POST /commerce/checkout/{session_id}/confirm": handleConfirm,
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
	case m == "POST" && path == "/commerce/checkout":
		return "POST /commerce/checkout"
	case m == "GET" && len(parts) == 3:
		return "GET /commerce/checkout/{session_id}"
	case m == "POST" && len(parts) == 4 && parts[3] == "confirm":
		return "POST /commerce/checkout/{session_id}/confirm"
	}
	return m + " " + path
}

// ─────────────────────────────────────────────
// POST /commerce/checkout
// Step 1: Create session, call tour-planner-ai for budget+conflict check,
//         then return the session (with AI validation) to the client.
// ─────────────────────────────────────────────

func handleInitiate(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body co.InitiateRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil || body.CartID == "" {
		return jsonError(http.StatusBadRequest, "cart_id is required"), nil
	}

	cart, err := co.GetCartSummary(ctx, db, body.CartID, userID.String())
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to load cart"), nil
	}
	if cart == nil {
		return jsonError(http.StatusNotFound, "cart not found"), nil
	}
	if cart.Status != "active" {
		return jsonError(http.StatusConflict, "cart is not active"), nil
	}
	if len(cart.Items) == 0 {
		return jsonError(http.StatusUnprocessableEntity, "cart is empty"), nil
	}

	session, err := co.CreateSession(ctx, db, cart.ID, userID.String(), cart.TotalAmount, cart.Currency, body.BillingDetails)
	if err != nil {
		log.Printf("CreateSession: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to create checkout session"), nil
	}

	// Call tour-planner-ai for budget validation and conflict detection.
	cartItemsForAI := cartItemsToAIFormat(cart.Items)
	tripContext := map[string]any{}
	if cart.TripPlanID != "" {
		tripContext["trip_plan_id"] = cart.TripPlanID
	}

	aiResult, _ := co.ValidateWithAI(ctx, cartItemsForAI, tripContext)
	_ = co.LogAIInteraction(ctx, db, userID.String(), cart.ID, "budget_check", cartItemsForAI, aiResult, aiResult.AgentIntent)

	newStatus := co.SessionStatusAIValidated
	if !aiResult.Valid {
		newStatus = co.SessionStatusFailed
	}
	_ = co.UpdateSessionStatus(ctx, db, session.ID, newStatus, aiResult)
	session.Status = newStatus
	session.AIValidation = aiResult

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       mustJSON(session),
	}, nil
}

// ─────────────────────────────────────────────
// GET /commerce/checkout/{session_id}
// ─────────────────────────────────────────────

func handleGetSession(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	sessionID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(sessionID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid session id"), nil
	}
	session, err := co.GetSession(ctx, db, sessionID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "db error"), nil
	}
	if session == nil {
		return jsonError(http.StatusNotFound, "session not found"), nil
	}
	if err = assertOwner(req, session.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}
	return jsonOK(session)
}

// ─────────────────────────────────────────────
// POST /commerce/checkout/{session_id}/confirm
// Step 2: Client has collected payment intent from Stripe; we create bookings
//         in tour-agent and promote the session to 'confirmed'.
// ─────────────────────────────────────────────

func handleConfirm(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	sessionID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(sessionID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid session id"), nil
	}

	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	var body co.ConfirmRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil || body.PaymentIntentID == "" {
		return jsonError(http.StatusBadRequest, "payment_intent_id is required"), nil
	}

	session, err := co.GetSession(ctx, db, sessionID)
	if err != nil || session == nil {
		return jsonError(http.StatusNotFound, "session not found"), nil
	}
	if err = assertOwner(req, session.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}
	if session.Status != co.SessionStatusAIValidated && session.Status != co.SessionStatusPaymentInitiated {
		return jsonError(http.StatusConflict, fmt.Sprintf("session status %s does not allow confirmation", session.Status)), nil
	}

	// Mark payment as initiated before calling tour-agent (idempotency guard).
	_ = co.UpdateSessionStatus(ctx, db, sessionID, co.SessionStatusPaymentInitiated, nil)

	cart, err := co.GetCartSummary(ctx, db, session.CartID, userID.String())
	if err != nil || cart == nil {
		return jsonError(http.StatusInternalServerError, "failed to load cart"), nil
	}

	// Create an order row so we have an ID for the item bookings.
	orderID, err := co.CreateOrder(ctx, db, session, cart, body.PaymentIntentID)
	if err != nil {
		log.Printf("CreateOrder: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to create order"), nil
	}

	// For each excursion item, create a booking in tour-agent.
	allConfirmed := true
	for _, item := range cart.Items {
		if item.ItemType != "excursion" {
			continue
		}
		bookingReq := co.TourAgentBookingRequest{
			UserID:       userID.String(),
			ExcursionID:  item.ItemID,
			TripPlanID:   cart.TripPlanID,
			TravelerDate: item.TravelerDate,
			Quantity:     item.Quantity,
			TotalAmount:  item.TotalPrice,
		}
		resp, bookErr := co.CreateTourAgentBooking(ctx, bookingReq)
		if bookErr != nil {
			log.Printf("CreateTourAgentBooking %s: %v", item.ItemID, bookErr)
			allConfirmed = false
			continue
		}
		_ = co.ConfirmOrderItem(ctx, db, orderID, item.ID, resp.BookingID)
	}

	finalStatus := "confirmed"
	if !allConfirmed {
		finalStatus = "partially_confirmed"
	}
	_ = co.SetOrderStatus(ctx, db, orderID, finalStatus)
	_ = co.UpdateSessionStatus(ctx, db, sessionID, co.SessionStatusConfirmed, nil)

	return jsonOK(map[string]any{
		"session_id":   sessionID,
		"order_id":     orderID,
		"status":       finalStatus,
		"total_amount": session.TotalAmount,
		"currency":     session.Currency,
	})
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

func cartItemsToAIFormat(items []co.CartItemSummary) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, it := range items {
		out = append(out, map[string]any{
			"item_type":    it.ItemType,
			"item_id":      it.ItemID,
			"quantity":     it.Quantity,
			"total_price":  it.TotalPrice,
			"traveler_date": it.TravelerDate,
		})
	}
	return out
}

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

func assertOwner(req events.APIGatewayProxyRequest, ownerID string) error {
	sub, ok := req.RequestContext.Authorizer["sub"].(string)
	if !ok || sub != ownerID {
		return fmt.Errorf("forbidden")
	}
	return nil
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
