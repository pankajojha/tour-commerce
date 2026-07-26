package cart

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

// Open returns a connected *sql.DB using DATABASE_URL from the environment.
func Open() (*sql.DB, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is not set")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

// GetCart fetches a cart with its items by cart ID.
func GetCart(ctx context.Context, db *sql.DB, cartID string) (*Cart, error) {
	c := &Cart{}
	var tripPlanID sql.NullString
	var expiresAt sql.NullTime
	var metaRaw []byte

	err := db.QueryRowContext(ctx, `
		SELECT id, user_id, trip_plan_id, status, currency, metadata, expires_at, created_at, updated_at
		FROM commerce_carts WHERE id = $1
	`, cartID).Scan(
		&c.ID, &c.UserID, &tripPlanID, &c.Status, &c.Currency,
		&metaRaw, &expiresAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetCart: %w", err)
	}
	if tripPlanID.Valid {
		c.TripPlanID = &tripPlanID.String
	}
	if expiresAt.Valid {
		c.ExpiresAt = &expiresAt.Time
	}
	_ = json.Unmarshal(metaRaw, &c.Metadata)

	items, err := getCartItems(ctx, db, cartID)
	if err != nil {
		return nil, err
	}
	c.Items = items
	c.TotalAmount = sumItems(items)
	return c, nil
}

// GetActiveCartForUser returns the user's active cart, or nil.
func GetActiveCartForUser(ctx context.Context, db *sql.DB, userID string) (*Cart, error) {
	var cartID string
	err := db.QueryRowContext(ctx,
		`SELECT id FROM commerce_carts WHERE user_id = $1 AND status = 'active' LIMIT 1`,
		userID,
	).Scan(&cartID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("GetActiveCartForUser: %w", err)
	}
	return GetCart(ctx, db, cartID)
}

// CreateCart inserts a new active cart and returns it.
func CreateCart(ctx context.Context, db *sql.DB, userID string, req CreateCartRequest) (*Cart, error) {
	cur := req.Currency
	if cur == "" {
		cur = "USD"
	}
	c := &Cart{}
	err := db.QueryRowContext(ctx, `
		INSERT INTO commerce_carts (user_id, trip_plan_id, currency)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, trip_plan_id, status, currency, metadata, expires_at, created_at, updated_at
	`, userID, nullStr(func() string {
		if req.TripPlanID != nil {
			return *req.TripPlanID
		}
		return ""
	}()), cur).Scan(
		&c.ID, &c.UserID, new(sql.NullString), &c.Status, &c.Currency,
		new([]byte), new(sql.NullTime), &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateCart: %w", err)
	}
	c.TripPlanID = req.TripPlanID
	c.Items = []CartItem{}
	return c, nil
}

// AddItem inserts a cart item and returns the updated cart.
func AddItem(ctx context.Context, db *sql.DB, cartID string, req AddItemRequest) (*CartItem, error) {
	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	totalPrice := req.UnitPrice * float64(req.Quantity)

	snapshotRaw, _ := json.Marshal(req.ItemSnapshot)
	aiCtxRaw, _ := json.Marshal(req.AIRecommendationContext)

	item := &CartItem{}
	err := db.QueryRowContext(ctx, `
		INSERT INTO commerce_cart_items
		    (cart_id, item_type, item_id, quantity, unit_price, total_price,
		     item_snapshot, ai_recommended, ai_recommendation_context, traveler_date)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, cart_id, item_type, item_id, quantity, unit_price, total_price,
		          item_snapshot, ai_recommended, ai_recommendation_context, traveler_date,
		          created_at::text, updated_at::text
	`, cartID, req.ItemType, req.ItemID, req.Quantity, req.UnitPrice, totalPrice,
		snapshotRaw, req.AIRecommended, aiCtxRaw, nullStr(func() string {
			if req.TravelerDate != nil {
				return *req.TravelerDate
			}
			return ""
		}()),
	).Scan(
		&item.ID, &item.CartID, &item.ItemType, &item.ItemID,
		&item.Quantity, &item.UnitPrice, &item.TotalPrice,
		new([]byte), &item.AIRecommended, new([]byte), new(sql.NullString),
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("AddItem: %w", err)
	}
	item.ItemSnapshot = req.ItemSnapshot
	item.TravelerDate = req.TravelerDate
	return item, nil
}

// RemoveItem deletes a cart item by ID.
func RemoveItem(ctx context.Context, db *sql.DB, cartID, itemID string) error {
	res, err := db.ExecContext(ctx,
		`DELETE FROM commerce_cart_items WHERE id = $1 AND cart_id = $2`,
		itemID, cartID,
	)
	if err != nil {
		return fmt.Errorf("RemoveItem: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("item not found")
	}
	return nil
}

// UpdateItemQuantity changes quantity and recomputes total_price.
func UpdateItemQuantity(ctx context.Context, db *sql.DB, cartID, itemID string, quantity int) (*CartItem, error) {
	item := &CartItem{}
	err := db.QueryRowContext(ctx, `
		UPDATE commerce_cart_items
		SET quantity    = $3,
		    total_price = unit_price * $3
		WHERE id = $1 AND cart_id = $2
		RETURNING id, cart_id, item_type, item_id, quantity, unit_price, total_price,
		          ai_recommended, traveler_date, created_at::text, updated_at::text
	`, itemID, cartID, quantity).Scan(
		&item.ID, &item.CartID, &item.ItemType, &item.ItemID,
		&item.Quantity, &item.UnitPrice, &item.TotalPrice,
		&item.AIRecommended, new(sql.NullString),
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item not found")
	}
	if err != nil {
		return nil, fmt.Errorf("UpdateItemQuantity: %w", err)
	}
	return item, nil
}

// MarkCartCheckedOut sets status = 'checked_out'.
func MarkCartCheckedOut(ctx context.Context, db *sql.DB, cartID string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE commerce_carts SET status = 'checked_out' WHERE id = $1`,
		cartID,
	)
	return err
}

// ─── private helpers ──────────────────────────────────────────────────────────

func getCartItems(ctx context.Context, db *sql.DB, cartID string) ([]CartItem, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, cart_id, item_type, item_id, quantity, unit_price, total_price,
		       item_snapshot, ai_recommended, ai_recommendation_context, traveler_date,
		       created_at::text, updated_at::text
		FROM commerce_cart_items WHERE cart_id = $1 ORDER BY created_at
	`, cartID)
	if err != nil {
		return nil, fmt.Errorf("getCartItems: %w", err)
	}
	defer rows.Close()

	var items []CartItem
	for rows.Next() {
		var item CartItem
		var snapshotRaw, aiCtxRaw []byte
		var travelerDate sql.NullString
		if err = rows.Scan(
			&item.ID, &item.CartID, &item.ItemType, &item.ItemID,
			&item.Quantity, &item.UnitPrice, &item.TotalPrice,
			&snapshotRaw, &item.AIRecommended, &aiCtxRaw, &travelerDate,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(snapshotRaw, &item.ItemSnapshot)
		_ = json.Unmarshal(aiCtxRaw, &item.AIRecommendationContext)
		if travelerDate.Valid {
			item.TravelerDate = &travelerDate.String
		}
		items = append(items, item)
	}
	if items == nil {
		items = []CartItem{}
	}
	return items, nil
}

func sumItems(items []CartItem) float64 {
	var total float64
	for _, it := range items {
		total += it.TotalPrice
	}
	return total
}

func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
