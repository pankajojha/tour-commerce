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

	cartpkg "github.com/explorer-agent/commerce-cart/cart"
)

var db *cartpkg.Client

func init() {
	var err error
	db, err = cartpkg.Open()
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
		"POST /commerce/cart":                     handleCreateCart,
		"GET /commerce/cart/active":               handleGetActiveCart,
		"GET /commerce/cart/{id}":                 handleGetCart,
		"POST /commerce/cart/{id}/items":          handleAddItem,
		"PUT /commerce/cart/{id}/items/{item_id}": handleUpdateItem,
		"DELETE /commerce/cart/{id}/items/{item_id}": handleRemoveItem,
	}
	key := matchRoute(req.HTTPMethod, req.Path)
	if h, ok := routes[key]; ok {
		return h(ctx, req)
	}
	return jsonError(http.StatusNotFound, "route not found"), nil
}

func matchRoute(method, path string) string {
	m := strings.ToUpper(method)
	switch {
	case m == "POST" && path == "/commerce/cart":
		return "POST /commerce/cart"
	case m == "GET" && path == "/commerce/cart/active":
		return "GET /commerce/cart/active"
	case m == "GET" && isCartRoot(path):
		return "GET /commerce/cart/{id}"
	case m == "POST" && strings.HasSuffix(path, "/items") && cartItemsPath(path):
		return "POST /commerce/cart/{id}/items"
	case m == "PUT" && cartItemPath(path):
		return "PUT /commerce/cart/{id}/items/{item_id}"
	case m == "DELETE" && cartItemPath(path):
		return "DELETE /commerce/cart/{id}/items/{item_id}"
	}
	return m + " " + path
}

func isCartRoot(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 2 && parts[0] == "commerce" && parts[1] != "cart"
}

func cartItemsPath(path string) bool {
	// /commerce/cart/{id}/items
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 4 && parts[3] == "items"
}

func cartItemPath(path string) bool {
	// /commerce/cart/{id}/items/{item_id}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 5 && parts[3] == "items"
}

// ─────────────────────────────────────────────
// POST /commerce/cart
// ─────────────────────────────────────────────

func handleCreateCart(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}

	existing, err := cartpkg.GetActiveCartForUser(ctx, db, userID.String())
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to check existing cart"), nil
	}
	if existing != nil {
		return jsonOK(existing)
	}

	var body cartpkg.CreateCartRequest
	if req.Body != "" {
		_ = json.Unmarshal([]byte(req.Body), &body)
	}

	c, err := cartpkg.CreateCart(ctx, db, userID.String(), body)
	if err != nil {
		log.Printf("CreateCart: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to create cart"), nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       mustJSON(c),
	}, nil
}

// ─────────────────────────────────────────────
// GET /commerce/cart/active
// ─────────────────────────────────────────────

func handleGetActiveCart(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	userID, err := subFromRequest(req)
	if err != nil {
		return jsonError(http.StatusUnauthorized, "missing or invalid subject claim"), nil
	}
	c, err := cartpkg.GetActiveCartForUser(ctx, db, userID.String())
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to fetch cart"), nil
	}
	if c == nil {
		return jsonError(http.StatusNotFound, "no active cart"), nil
	}
	return jsonOK(c)
}

// ─────────────────────────────────────────────
// GET /commerce/cart/{id}
// ─────────────────────────────────────────────

func handleGetCart(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cartID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(cartID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid cart id"), nil
	}
	c, err := cartpkg.GetCart(ctx, db, cartID)
	if err != nil {
		return jsonError(http.StatusInternalServerError, "failed to fetch cart"), nil
	}
	if c == nil {
		return jsonError(http.StatusNotFound, "cart not found"), nil
	}
	if err = assertOwner(req, c.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}
	return jsonOK(c)
}

// ─────────────────────────────────────────────
// POST /commerce/cart/{id}/items
// ─────────────────────────────────────────────

func handleAddItem(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cartID := pathSegment(req.Path, 2)
	if _, err := uuid.Parse(cartID); err != nil {
		return jsonError(http.StatusBadRequest, "invalid cart id"), nil
	}

	c, err := cartpkg.GetCart(ctx, db, cartID)
	if err != nil || c == nil {
		return jsonError(http.StatusNotFound, "cart not found"), nil
	}
	if err = assertOwner(req, c.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}
	if c.Status != cartpkg.CartStatusActive {
		return jsonError(http.StatusConflict, "cart is not active"), nil
	}

	var body cartpkg.AddItemRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.ItemType == "" || body.ItemID == "" {
		return jsonError(http.StatusBadRequest, "item_type and item_id are required"), nil
	}
	if body.UnitPrice < 0 {
		return jsonError(http.StatusBadRequest, "unit_price must be non-negative"), nil
	}

	item, err := cartpkg.AddItem(ctx, db, cartID, body)
	if err != nil {
		log.Printf("AddItem: %v", err)
		return jsonError(http.StatusInternalServerError, "failed to add item"), nil
	}
	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusCreated,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       mustJSON(item),
	}, nil
}

// ─────────────────────────────────────────────
// PUT /commerce/cart/{id}/items/{item_id}
// ─────────────────────────────────────────────

func handleUpdateItem(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cartID := pathSegment(req.Path, 2)
	itemID := pathSegment(req.Path, 4)

	c, err := cartpkg.GetCart(ctx, db, cartID)
	if err != nil || c == nil {
		return jsonError(http.StatusNotFound, "cart not found"), nil
	}
	if err = assertOwner(req, c.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}

	var body cartpkg.UpdateItemRequest
	if err = json.Unmarshal([]byte(req.Body), &body); err != nil {
		return jsonError(http.StatusBadRequest, "invalid request body"), nil
	}
	if body.Quantity == nil {
		return jsonError(http.StatusBadRequest, "quantity is required"), nil
	}
	if *body.Quantity <= 0 {
		return jsonError(http.StatusBadRequest, "quantity must be positive"), nil
	}

	item, err := cartpkg.UpdateItemQuantity(ctx, db, cartID, itemID, *body.Quantity)
	if err != nil {
		if err.Error() == "item not found" {
			return jsonError(http.StatusNotFound, "item not found"), nil
		}
		return jsonError(http.StatusInternalServerError, "failed to update item"), nil
	}
	return jsonOK(item)
}

// ─────────────────────────────────────────────
// DELETE /commerce/cart/{id}/items/{item_id}
// ─────────────────────────────────────────────

func handleRemoveItem(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	cartID := pathSegment(req.Path, 2)
	itemID := pathSegment(req.Path, 4)

	c, err := cartpkg.GetCart(ctx, db, cartID)
	if err != nil || c == nil {
		return jsonError(http.StatusNotFound, "cart not found"), nil
	}
	if err = assertOwner(req, c.UserID); err != nil {
		return jsonError(http.StatusForbidden, "forbidden"), nil
	}

	if err = cartpkg.RemoveItem(ctx, db, cartID, itemID); err != nil {
		if err.Error() == "item not found" {
			return jsonError(http.StatusNotFound, "item not found"), nil
		}
		return jsonError(http.StatusInternalServerError, "failed to remove item"), nil
	}
	return events.APIGatewayProxyResponse{StatusCode: http.StatusNoContent}, nil
}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

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
