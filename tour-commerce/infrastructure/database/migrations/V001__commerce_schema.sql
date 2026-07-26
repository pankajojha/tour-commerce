-- ============================================================================
-- Tour Commerce - PostgreSQL Database Schema
-- ============================================================================
-- Version: 1.0
-- Database: PostgreSQL 14+
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- SECTION 1: CARTS
-- ============================================================================

CREATE TABLE commerce_carts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID NOT NULL,
    trip_plan_id    UUID,                           -- optional link to tour-agent trip_plans
    status          VARCHAR(50) NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'checked_out', 'abandoned', 'expired')),
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    metadata        JSONB NOT NULL DEFAULT '{}',
    expires_at      TIMESTAMP WITH TIME ZONE,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_carts IS 'Shopping carts; one active cart per user at a time';
COMMENT ON COLUMN commerce_carts.trip_plan_id IS 'References trip_plans.id in tour-agent DB (cross-service FK, not enforced)';

CREATE INDEX idx_commerce_carts_user_status ON commerce_carts (user_id, status);
CREATE INDEX idx_commerce_carts_trip_plan   ON commerce_carts (trip_plan_id) WHERE trip_plan_id IS NOT NULL;

CREATE TABLE commerce_cart_items (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cart_id                     UUID NOT NULL REFERENCES commerce_carts(id) ON DELETE CASCADE,
    item_type                   VARCHAR(50) NOT NULL
                                    CHECK (item_type IN ('excursion', 'hotel', 'flight', 'restaurant', 'cruise', 'package')),
    item_id                     VARCHAR(255) NOT NULL,      -- external ID in tour-agent or supplier
    quantity                    INTEGER NOT NULL DEFAULT 1
                                    CHECK (quantity > 0 AND quantity <= 100),
    unit_price                  DECIMAL(10,2) NOT NULL CHECK (unit_price >= 0),
    total_price                 DECIMAL(10,2) NOT NULL CHECK (total_price >= 0),
    item_snapshot               JSONB NOT NULL DEFAULT '{}', -- item details captured at add time
    ai_recommended              BOOLEAN NOT NULL DEFAULT FALSE,
    ai_recommendation_context   JSONB,                       -- agent intent + reasoning
    traveler_date               DATE,                        -- when the item is scheduled
    created_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at                  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_cart_items IS 'Line items inside a cart, covering all bookable item types';
COMMENT ON COLUMN commerce_cart_items.item_snapshot IS 'Frozen copy of item details at the time of add, so price/title changes do not mutate the cart';
COMMENT ON COLUMN commerce_cart_items.ai_recommended IS 'TRUE when tour-planner-ai suggested this item';

CREATE INDEX idx_cart_items_cart   ON commerce_cart_items (cart_id);
CREATE INDEX idx_cart_items_type   ON commerce_cart_items (item_type, item_id);

-- ============================================================================
-- SECTION 2: CHECKOUT SESSIONS
-- ============================================================================

CREATE TABLE commerce_checkout_sessions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cart_id         UUID NOT NULL REFERENCES commerce_carts(id),
    user_id         UUID NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'ai_validated', 'payment_initiated', 'confirmed', 'failed', 'expired')),
    ai_validation   JSONB,              -- budget check + conflict detection from tour-planner-ai
    billing_details JSONB NOT NULL DEFAULT '{}',
    total_amount    DECIMAL(10,2) NOT NULL CHECK (total_amount >= 0),
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    expires_at      TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_checkout_sessions IS 'Short-lived checkout sessions; validated by tour-planner-ai before payment';
COMMENT ON COLUMN commerce_checkout_sessions.ai_validation IS 'Snapshot of tour-planner-ai budget/conflict response used to decide whether to allow payment';

CREATE INDEX idx_checkout_sessions_cart   ON commerce_checkout_sessions (cart_id);
CREATE INDEX idx_checkout_sessions_user   ON commerce_checkout_sessions (user_id, status);

-- ============================================================================
-- SECTION 3: ORDERS
-- ============================================================================

CREATE TABLE commerce_orders (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    checkout_session_id UUID REFERENCES commerce_checkout_sessions(id),
    cart_id             UUID NOT NULL REFERENCES commerce_carts(id),
    user_id             UUID NOT NULL,
    trip_plan_id        UUID,
    status              VARCHAR(50) NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'confirmed', 'partially_confirmed', 'failed', 'cancelled', 'refunded')),
    total_amount        DECIMAL(10,2) NOT NULL CHECK (total_amount >= 0),
    refunded_amount     DECIMAL(10,2) NOT NULL DEFAULT 0 CHECK (refunded_amount >= 0),
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    payment_intent_id   VARCHAR(255),
    billing_details     JSONB NOT NULL DEFAULT '{}',
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_orders IS 'Confirmed purchase orders, one per checkout session';
COMMENT ON COLUMN commerce_orders.trip_plan_id IS 'Cross-service reference to tour-agent trip_plans.id';

CREATE INDEX idx_orders_user        ON commerce_orders (user_id, status);
CREATE INDEX idx_orders_trip_plan   ON commerce_orders (trip_plan_id) WHERE trip_plan_id IS NOT NULL;
CREATE INDEX idx_orders_payment     ON commerce_orders (payment_intent_id) WHERE payment_intent_id IS NOT NULL;

CREATE TABLE commerce_order_items (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id                UUID NOT NULL REFERENCES commerce_orders(id) ON DELETE CASCADE,
    item_type               VARCHAR(50) NOT NULL
                                CHECK (item_type IN ('excursion', 'hotel', 'flight', 'restaurant', 'cruise', 'package')),
    item_id                 VARCHAR(255) NOT NULL,
    tour_agent_booking_id   VARCHAR(255),           -- booking ID created in tour-agent after confirmation
    quantity                INTEGER NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price              DECIMAL(10,2) NOT NULL CHECK (unit_price >= 0),
    total_price             DECIMAL(10,2) NOT NULL CHECK (total_price >= 0),
    item_snapshot           JSONB NOT NULL DEFAULT '{}',
    status                  VARCHAR(50) NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending', 'confirmed', 'failed', 'cancelled', 'refunded')),
    failure_reason          TEXT,
    traveler_date           DATE,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_order_items IS 'Individual bookable items within an order, each confirmed independently with tour-agent';
COMMENT ON COLUMN commerce_order_items.tour_agent_booking_id IS 'The booking ID returned by tour-agent after POST /bookings succeeds';

CREATE INDEX idx_order_items_order         ON commerce_order_items (order_id);
CREATE INDEX idx_order_items_booking       ON commerce_order_items (tour_agent_booking_id) WHERE tour_agent_booking_id IS NOT NULL;

-- ============================================================================
-- SECTION 4: PAYMENTS
-- ============================================================================

CREATE TABLE commerce_payments (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id            UUID NOT NULL REFERENCES commerce_orders(id),
    provider            VARCHAR(50) NOT NULL DEFAULT 'stripe'
                            CHECK (provider IN ('stripe', 'paypal', 'internal')),
    provider_payment_id VARCHAR(255) NOT NULL,   -- Stripe PaymentIntent ID
    provider_charge_id  VARCHAR(255),            -- Stripe Charge ID (set after capture)
    amount              DECIMAL(10,2) NOT NULL CHECK (amount > 0),
    refunded_amount     DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    status              VARCHAR(50) NOT NULL
                            CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'refunded', 'partially_refunded')),
    payment_method_type VARCHAR(50),             -- card, apple_pay, google_pay, etc.
    last4               VARCHAR(4),              -- masked card last four digits
    failure_code        VARCHAR(100),
    failure_message     TEXT,
    stripe_metadata     JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_payments IS 'Payment records, one-to-one with orders for initial charges; additional rows for refunds';

CREATE UNIQUE INDEX idx_payments_provider_id ON commerce_payments (provider, provider_payment_id);
CREATE INDEX idx_payments_order              ON commerce_payments (order_id);

-- ============================================================================
-- SECTION 5: AI COMMERCE INTERACTIONS
-- ============================================================================

CREATE TABLE commerce_ai_interactions (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL,
    cart_id             UUID REFERENCES commerce_carts(id) ON DELETE SET NULL,
    trip_plan_id        UUID,
    interaction_type    VARCHAR(50) NOT NULL
                            CHECK (interaction_type IN ('recommendation', 'budget_check', 'upsell', 'conflict_check', 'itinerary_summary')),
    request_payload     JSONB NOT NULL,
    response_payload    JSONB,
    agent_intent        VARCHAR(100),            -- which specialist agent tour-planner-ai routed to
    item_applied_count  INTEGER NOT NULL DEFAULT 0,  -- how many recommended items user added to cart
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE commerce_ai_interactions IS 'Audit log of all tour-planner-ai calls made during the commerce flow';
COMMENT ON COLUMN commerce_ai_interactions.item_applied_count IS 'Number of AI-recommended items the user actually added to the cart';

CREATE INDEX idx_ai_interactions_user ON commerce_ai_interactions (user_id, interaction_type);
CREATE INDEX idx_ai_interactions_cart ON commerce_ai_interactions (cart_id) WHERE cart_id IS NOT NULL;

-- ============================================================================
-- SECTION 6: updated_at triggers
-- ============================================================================

CREATE OR REPLACE FUNCTION commerce_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_commerce_carts_updated_at
    BEFORE UPDATE ON commerce_carts
    FOR EACH ROW EXECUTE FUNCTION commerce_set_updated_at();

CREATE TRIGGER trg_commerce_cart_items_updated_at
    BEFORE UPDATE ON commerce_cart_items
    FOR EACH ROW EXECUTE FUNCTION commerce_set_updated_at();

CREATE TRIGGER trg_commerce_checkout_sessions_updated_at
    BEFORE UPDATE ON commerce_checkout_sessions
    FOR EACH ROW EXECUTE FUNCTION commerce_set_updated_at();

CREATE TRIGGER trg_commerce_orders_updated_at
    BEFORE UPDATE ON commerce_orders
    FOR EACH ROW EXECUTE FUNCTION commerce_set_updated_at();

CREATE TRIGGER trg_commerce_order_items_updated_at
    BEFORE UPDATE ON commerce_order_items
    FOR EACH ROW EXECUTE FUNCTION commerce_set_updated_at();

CREATE TRIGGER trg_commerce_payments_updated_at
    BEFORE UPDATE ON commerce_payments
    FOR EACH ROW EXECUTE FUNCTION commerce_set_updated_at();
