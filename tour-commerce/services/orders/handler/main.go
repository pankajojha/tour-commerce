package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/google/uuid"

	orderpkg "github.com/explorer-agent/commerce-orders/orders"
)

var db *sql.DB

func init() {
	var err error
	db, err = orderpkg.Open()
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
}

type routeHandler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	routes := map[string]routeHandler{
		"GET /commerce/orders":              handleListOrders,
		"GET /commerce/orders/{id}":         handleGetOrder,
		"POST /commerce/orders/{id}/cancel": handleCancelOrder,
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
	case m == "GET" && len(parts) == 2 && parts[1] == "orders":
		return "GET /commerce/orders"
	case m == "GET" && len(parts) == 3:
		return "GET /commerce/orders/{id}"
	case m == "POST" && len(parts) == 4 && parts[3] == "cancel":
		return "POST /commerce/orders/{id}/cancel"
	}
	return m + " " + path
}

// GET /commerce/orders
func handleListOrders(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	limit, _ := strconv.Atoi(req.QueryStringParameters["limit"])
	offset, _ := strconv.Atoi(req.QueryStringParameters["offset"])

	orders, total, err := orderpkg.ListOrders(ctx, db, userID.String(), limit, offset)
	if err != nil {
		log.Printf("ListOrders: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to list orders"), nil
	}
	return jsonOK(orderpkg.ListOrdersResponse{Orders: orders, Total: total})
}

// GET /commerce/orders/{id}
func handleGetOrder(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	orderID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(orderID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid order id"), nil
	}

	o, err := orderpkg.GetOrder(ctx, db, orderID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "db error"), nil
	}
	if o == nil {
		return jsonError(http.StatusNotFound, "order not found"), nil
	}
	if err = assertOwner(req, o.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}
	return jsonOK(o)
}

// POST /commerce/orders/{id}/cancel
func handleCancelOrder(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	orderID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(orderID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid order id"), nil
	}

	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	if err = orderpkg.CancelOrder(ctx, db, orderID, userID.String()); err != nil {
		if strings.Contains(err.Error(), "not found or cannot") {
			return jsonError(http.StatusConflict, err.Error()), nil
		}
		return jsonError(http.StatusInternalServerError, "failed to cancel order"), nil
	}
	return jsonOK(map[string]string{"order_id": orderID, "status": "cancelled"})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

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
