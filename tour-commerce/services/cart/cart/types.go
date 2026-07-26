package cart

import "time"

// ItemType enumerates the product categories that can appear in a cart.
type ItemType string

const (
	ItemTypeExcursion  ItemType = "excursion"
	ItemTypeHotel      ItemType = "hotel"
	ItemTypeFlight     ItemType = "flight"
	ItemTypeRestaurant ItemType = "restaurant"
	ItemTypeCruise     ItemType = "cruise"
	ItemTypePackage    ItemType = "package"
)

// CartStatus is the lifecycle state of a cart.
type CartStatus string

const (
	CartStatusActive      CartStatus = "active"
	CartStatusCheckedOut  CartStatus = "checked_out"
	CartStatusAbandoned   CartStatus = "abandoned"
	CartStatusExpired     CartStatus = "expired"
)

// Cart represents a commerce_carts row.
type Cart struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	TripPlanID  *string    `json:"trip_plan_id,omitempty"`
	Status      CartStatus `json:"status"`
	Currency    string     `json:"currency"`
	Metadata    any        `json:"metadata"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Items       []CartItem `json:"items,omitempty"`
	TotalAmount float64    `json:"total_amount,omitempty"`
}

// CartItem represents a commerce_cart_items row.
type CartItem struct {
	ID                       string   `json:"id"`
	CartID                   string   `json:"cart_id"`
	ItemType                 ItemType `json:"item_type"`
	ItemID                   string   `json:"item_id"`
	Quantity                 int      `json:"quantity"`
	UnitPrice                float64  `json:"unit_price"`
	TotalPrice               float64  `json:"total_price"`
	ItemSnapshot             any      `json:"item_snapshot"`
	AIRecommended            bool     `json:"ai_recommended"`
	AIRecommendationContext  any      `json:"ai_recommendation_context,omitempty"`
	TravelerDate             *string  `json:"traveler_date,omitempty"`
	CreatedAt                string   `json:"created_at"`
	UpdatedAt                string   `json:"updated_at"`
}

// AddItemRequest is the body for POST /commerce/cart/{id}/items.
type AddItemRequest struct {
	ItemType                ItemType `json:"item_type"`
	ItemID                  string   `json:"item_id"`
	Quantity                int      `json:"quantity"`
	UnitPrice               float64  `json:"unit_price"`
	ItemSnapshot            any      `json:"item_snapshot"`
	AIRecommended           bool     `json:"ai_recommended,omitempty"`
	AIRecommendationContext any      `json:"ai_recommendation_context,omitempty"`
	TravelerDate            *string  `json:"traveler_date,omitempty"`
}

// UpdateItemRequest is the body for PUT /commerce/cart/{id}/items/{item_id}.
type UpdateItemRequest struct {
	Quantity *int     `json:"quantity,omitempty"`
	UnitPrice *float64 `json:"unit_price,omitempty"`
}

// CreateCartRequest is the body for POST /commerce/cart.
type CreateCartRequest struct {
	TripPlanID *string `json:"trip_plan_id,omitempty"`
	Currency   string  `json:"currency,omitempty"`
}
