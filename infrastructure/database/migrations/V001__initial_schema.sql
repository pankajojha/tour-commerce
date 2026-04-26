-- ============================================================================
-- Explorer Excursion Booking Platform - PostgreSQL Database Schema
-- ============================================================================
-- Description: Complete database schema for explorer excursion booking platform
-- Version: 1.0
-- Database: PostgreSQL 14+
-- Features: UUID primary keys, JSONB columns, triggers, indexes, pg_cron
-- ============================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_cron";

-- ============================================================================
-- SECTION 1: USERS & AUTHENTICATION
-- ============================================================================

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cognito_user_id VARCHAR(255) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    phone_number VARCHAR(20),
    phone_number_verified BOOLEAN DEFAULT FALSE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    date_of_birth DATE,
    country VARCHAR(2),
    profile_image_url TEXT,
    auth_provider VARCHAR(50) NOT NULL CHECK (auth_provider IN ('Email', 'Facebook', 'Google', 'Instagram', 'TikTok')),
    preferences JSONB DEFAULT '{"currency": "USD", "language": "en", "notifications": {"email": true, "sms": false, "push": true}}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE
);

COMMENT ON TABLE users IS 'User accounts with authentication and profile information';
COMMENT ON COLUMN users.cognito_user_id IS 'AWS Cognito User Pool ID';
COMMENT ON COLUMN users.preferences IS 'User preferences stored as JSONB (currency, language, notifications)';

-- Social profiles table
CREATE TABLE social_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('Facebook', 'Google', 'Instagram', 'TikTok')),
    provider_id VARCHAR(255) NOT NULL,
    profile_url TEXT,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, provider)
);

COMMENT ON TABLE social_profiles IS 'Social media authentication profiles linked to users';

-- Sessions table
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE sessions IS 'User session tracking for JWT tokens';

-- ============================================================================
-- SECTION 2: DESTINATIONS & CATEGORIES
-- ============================================================================

-- Destinations table
CREATE TABLE destinations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    country VARCHAR(100) NOT NULL,
    port_code VARCHAR(10),
    airport_code VARCHAR(10),
    timezone VARCHAR(100) NOT NULL,
    geo_coordinates JSONB NOT NULL,
    description TEXT,
    image_urls JSONB DEFAULT '[]',
    popular_categories JSONB DEFAULT '[]',
    active_excursion_count INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_rating CHECK (average_rating >= 0 AND average_rating <= 5),
    CONSTRAINT valid_geo_coordinates CHECK (jsonb_typeof(geo_coordinates) = 'object')
);

COMMENT ON TABLE destinations IS 'Destination locations for excursions (ports, airports, cities)';
COMMENT ON COLUMN destinations.timezone IS 'IANA timezone identifier (e.g., America/Cancun)';
COMMENT ON COLUMN destinations.geo_coordinates IS 'JSONB: {"latitude": 21.1619, "longitude": -86.8515}';

-- Categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    icon_url TEXT,
    parent_category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    display_order INTEGER DEFAULT 0,
    excursion_count INTEGER DEFAULT 0,
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE categories IS 'Excursion categories and subcategories';

-- ============================================================================
-- SECTION 3: SUPPLIERS
-- ============================================================================

-- Suppliers table
CREATE TABLE suppliers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(20) NOT NULL,
    address TEXT,
    contract_type VARCHAR(50) NOT NULL CHECK (contract_type IN ('Commission', 'Net Rate', 'Markup')),
    commission_percentage DECIMAL(5,2) DEFAULT 0.00,
    payment_terms TEXT,
    bank_details JSONB,
    active_status BOOLEAN DEFAULT TRUE,
    rating DECIMAL(3,2) DEFAULT 0.00,
    excursion_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_commission CHECK (commission_percentage >= 0 AND commission_percentage <= 100),
    CONSTRAINT valid_supplier_rating CHECK (rating >= 0 AND rating <= 5)
);

COMMENT ON TABLE suppliers IS 'Explorer operators and excursion suppliers';
COMMENT ON COLUMN suppliers.bank_details IS 'Encrypted bank account information stored as JSONB';

-- ============================================================================
-- SECTION 4: EXCURSIONS & SCHEDULES
-- ============================================================================

-- Excursions table
CREATE TABLE excursions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
    supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE CASCADE,
    duration_minutes INTEGER NOT NULL,
    difficulty_level VARCHAR(50) NOT NULL CHECK (difficulty_level IN ('Easy', 'Moderate', 'Challenging', 'Extreme')),
    meeting_point TEXT NOT NULL,
    meeting_point_coordinates JSONB,
    cancellation_policy TEXT NOT NULL,
    base_price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    active_status BOOLEAN DEFAULT TRUE,
    categories JSONB DEFAULT '[]',
    image_urls JSONB DEFAULT '[]',
    video_url TEXT,
    inclusions JSONB DEFAULT '[]',
    exclusions JSONB DEFAULT '[]',
    what_to_bring JSONB DEFAULT '[]',
    accessibility JSONB DEFAULT '{"wheelchairAccessible": false, "mobilityAssistance": false}',
    min_age INTEGER,
    max_group_size INTEGER NOT NULL,
    languages JSONB DEFAULT '["en"]',
    rating DECIMAL(3,2) DEFAULT 0.00,
    review_count INTEGER DEFAULT 0,
    booking_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_duration CHECK (duration_minutes > 0 AND duration_minutes <= 1440),
    CONSTRAINT valid_price CHECK (base_price > 0),
    CONSTRAINT valid_excursion_rating CHECK (rating >= 0 AND rating <= 5),
    CONSTRAINT valid_max_group_size CHECK (max_group_size > 0 AND max_group_size <= 500)
);

COMMENT ON TABLE excursions IS 'Explorer excursions and activities';
COMMENT ON COLUMN excursions.duration_minutes IS 'Duration in minutes (max 1440 = 24 hours)';
COMMENT ON COLUMN excursions.accessibility IS 'JSONB: accessibility features and notes';

-- Excursion schedules table
CREATE TABLE excursion_schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    excursion_id UUID NOT NULL REFERENCES excursions(id) ON DELETE CASCADE,
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    capacity INTEGER NOT NULL,
    operating_days JSONB DEFAULT '["Mon","Tue","Wed","Thu","Fri","Sat","Sun"]',
    blackout_dates JSONB DEFAULT '[]',
    cutoff_hours INTEGER NOT NULL DEFAULT 4,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_capacity CHECK (capacity > 0 AND capacity <= 500),
    CONSTRAINT valid_cutoff CHECK (cutoff_hours >= 1 AND cutoff_hours <= 72),
    CONSTRAINT valid_time_range CHECK (end_time > start_time)
);

COMMENT ON TABLE excursion_schedules IS 'Operating schedules for excursions';
COMMENT ON COLUMN excursion_schedules.operating_days IS 'JSONB array: ["Mon", "Tue", "Wed", ...]';
COMMENT ON COLUMN excursion_schedules.blackout_dates IS 'JSONB array: ["2024-12-25", "2024-01-01"]';
COMMENT ON COLUMN excursion_schedules.cutoff_hours IS 'Hours before departure when booking closes';

-- ============================================================================
-- SECTION 5: TRAVEL (CRUISE & FLIGHT)
-- ============================================================================

-- Cruise port calls table
CREATE TABLE cruise_port_calls (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    cruise_line VARCHAR(100) NOT NULL,
    ship_name VARCHAR(100) NOT NULL,
    sailing_id VARCHAR(100) NOT NULL,
    destination_id UUID NOT NULL REFERENCES destinations(id) ON DELETE CASCADE,
    arrival_time TIMESTAMP WITH TIME ZONE NOT NULL,
    departure_time TIMESTAMP WITH TIME ZONE NOT NULL,
    date DATE NOT NULL,
    port_code VARCHAR(10) NOT NULL,
    timezone VARCHAR(100) NOT NULL,
    passenger_capacity INTEGER,
    status VARCHAR(50) DEFAULT 'Scheduled' CHECK (status IN ('Scheduled', 'Delayed', 'Cancelled', 'Completed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_cruise_times CHECK (departure_time > arrival_time),
    CONSTRAINT valid_passenger_capacity CHECK (passenger_capacity IS NULL OR passenger_capacity > 0)
);

COMMENT ON TABLE cruise_port_calls IS 'Cruise ship port calls and schedules';
COMMENT ON COLUMN cruise_port_calls.timezone IS 'IANA timezone for the port';

-- Flight arrivals table
CREATE TABLE flight_arrivals (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    airline VARCHAR(100) NOT NULL,
    flight_number VARCHAR(20) NOT NULL,
    arrival_airport VARCHAR(10) NOT NULL,
    arrival_time TIMESTAMP WITH TIME ZONE NOT NULL,
    date DATE NOT NULL,
    timezone VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'Scheduled' CHECK (status IN ('Scheduled', 'Delayed', 'Landed', 'Cancelled')),
    gate VARCHAR(10),
    terminal VARCHAR(10),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE flight_arrivals IS 'Flight arrival schedules for explorer eligibility';

-- ============================================================================
-- SECTION 6: INVENTORY MANAGEMENT
-- ============================================================================

-- Inventory records table
CREATE TABLE inventory_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    excursion_id UUID NOT NULL REFERENCES excursions(id) ON DELETE CASCADE,
    excursion_schedule_id UUID NOT NULL REFERENCES excursion_schedules(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_capacity INTEGER NOT NULL,
    booked_spots INTEGER DEFAULT 0,
    held_spots INTEGER DEFAULT 0,
    available_spots INTEGER GENERATED ALWAYS AS (total_capacity - booked_spots - held_spots) STORED,
    is_blackout BOOLEAN DEFAULT FALSE,
    blackout_reason TEXT,
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    version INTEGER DEFAULT 1,
    CONSTRAINT valid_inventory_capacity CHECK (total_capacity > 0),
    CONSTRAINT valid_booked_spots CHECK (booked_spots >= 0),
    CONSTRAINT valid_held_spots CHECK (held_spots >= 0),
    CONSTRAINT no_overbooking CHECK (booked_spots + held_spots <= total_capacity),
    UNIQUE(excursion_id, date)
);

COMMENT ON TABLE inventory_records IS 'Daily inventory tracking for excursions';
COMMENT ON COLUMN inventory_records.available_spots IS 'Computed column: total_capacity - booked_spots - held_spots';
COMMENT ON COLUMN inventory_records.version IS 'Optimistic locking version number';

-- ============================================================================
-- SECTION 7: PRICING CONFIGURATIONS
-- ============================================================================

-- Pricing configurations table
CREATE TABLE pricing_configurations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    excursion_id UUID NOT NULL REFERENCES excursions(id) ON DELETE CASCADE,
    pricing_type VARCHAR(50) NOT NULL CHECK (pricing_type IN ('Fixed', 'Tiered', 'Dynamic')),
    base_price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    tiers JSONB DEFAULT '[]',
    dynamic_pricing_enabled BOOLEAN DEFAULT FALSE,
    dynamic_factors JSONB DEFAULT '{}',
    seasonal_pricing JSONB DEFAULT '[]',
    cruise_line_pricing JSONB DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_pricing_base_price CHECK (base_price > 0),
    UNIQUE(excursion_id)
);

COMMENT ON TABLE pricing_configurations IS 'Pricing rules and dynamic pricing configurations';
COMMENT ON COLUMN pricing_configurations.tiers IS 'JSONB: [{"passengerType": "Adult", "ageRange": "12+", "price": 100}]';
COMMENT ON COLUMN pricing_configurations.dynamic_factors IS 'JSONB: demand, seasonal, cruise line multipliers';
COMMENT ON COLUMN pricing_configurations.seasonal_pricing IS 'JSONB: [{"season": "High", "startDate": "12-01", "endDate": "03-31", "multiplier": 1.3}]';

-- ============================================================================
-- SECTION 8: BOOKINGS
-- ============================================================================

-- Bookings table
CREATE TABLE bookings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    excursion_id UUID NOT NULL REFERENCES excursions(id) ON DELETE CASCADE,
    excursion_schedule_id UUID NOT NULL REFERENCES excursion_schedules(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending' CHECK (status IN ('Pending', 'PaymentHold', 'Confirmed', 'Cancelled', 'Completed', 'NoShow', 'Refunded')),
    passengers JSONB NOT NULL,
    total_passengers INTEGER NOT NULL,
    total_price DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    price_breakdown JSONB DEFAULT '[]',
    payment_status VARCHAR(50) DEFAULT 'Pending' CHECK (payment_status IN ('Pending', 'Processing', 'Paid', 'Failed', 'Refunded', 'PartiallyRefunded')),
    payment_intent_id VARCHAR(255),
    transaction_id VARCHAR(255),
    voucher_url TEXT,
    qr_code TEXT,
    emergency_contact JSONB NOT NULL,
    special_requests TEXT,
    cruise_port_call_id UUID REFERENCES cruise_port_calls(id) ON DELETE SET NULL,
    flight_arrival_id UUID REFERENCES flight_arrivals(id) ON DELETE SET NULL,
    cruise_info JSONB,
    flight_info JSONB,
    check_in_status VARCHAR(50) DEFAULT 'Pending' CHECK (check_in_status IN ('Pending', 'CheckedIn', 'NoShow')),
    check_in_time TIMESTAMP WITH TIME ZONE,
    cancellation_reason TEXT,
    cancellation_fee DECIMAL(10,2),
    refund_amount DECIMAL(10,2),
    refund_status VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    confirmed_at TIMESTAMP WITH TIME ZONE,
    cancelled_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_total_price CHECK (total_price > 0),
    CONSTRAINT valid_total_passengers CHECK (total_passengers > 0),
    CONSTRAINT valid_cancellation_fee CHECK (cancellation_fee IS NULL OR cancellation_fee >= 0),
    CONSTRAINT valid_refund_amount CHECK (refund_amount IS NULL OR refund_amount >= 0)
);

COMMENT ON TABLE bookings IS 'Customer bookings for excursions';
COMMENT ON COLUMN bookings.passengers IS 'JSONB array of passenger details';
COMMENT ON COLUMN bookings.emergency_contact IS 'JSONB: {"name": "...", "phoneNumber": "...", "relationship": "..."}';
COMMENT ON COLUMN bookings.cruise_info IS 'JSONB: cruise details if applicable';
COMMENT ON COLUMN bookings.flight_info IS 'JSONB: flight details if applicable';

-- ============================================================================
-- SECTION 9: PAYMENTS & REFUNDS
-- ============================================================================

-- Payments table
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    payment_intent_id VARCHAR(255) NOT NULL UNIQUE,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    payment_method VARCHAR(50),
    status VARCHAR(50) NOT NULL CHECK (status IN ('Pending', 'Processing', 'Paid', 'Failed', 'Refunded', 'PartiallyRefunded')),
    transaction_id VARCHAR(255),
    failure_reason TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    paid_at TIMESTAMP WITH TIME ZONE,
    refunded_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_payment_amount CHECK (amount > 0)
);

COMMENT ON TABLE payments IS 'Payment transactions for bookings';

-- Refunds table
CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'USD',
    reason TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'Pending' CHECK (status IN ('Pending', 'Processing', 'Completed', 'Failed')),
    refund_id VARCHAR(255),
    estimated_arrival DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_refund_amount CHECK (amount > 0)
);

COMMENT ON TABLE refunds IS 'Refund transactions for cancelled bookings';

-- ============================================================================
-- SECTION 10: REVIEWS
-- ============================================================================

-- Reviews table
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    excursion_id UUID NOT NULL REFERENCES excursions(id) ON DELETE CASCADE,
    booking_id UUID NOT NULL REFERENCES bookings(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL,
    title VARCHAR(255),
    comment TEXT,
    photos JSONB DEFAULT '[]',
    helpful INTEGER DEFAULT 0,
    verified BOOLEAN DEFAULT FALSE,
    supplier_response JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_review_rating CHECK (rating >= 1 AND rating <= 5),
    CONSTRAINT valid_helpful_count CHECK (helpful >= 0),
    UNIQUE(booking_id)
);

COMMENT ON TABLE reviews IS 'Customer reviews for excursions';
COMMENT ON COLUMN reviews.verified IS 'TRUE if review is from verified booking';
COMMENT ON COLUMN reviews.supplier_response IS 'JSONB: {"supplierId": "...", "response": "...", "respondedAt": "..."}';

-- ============================================================================
-- SECTION 11: AUDIT LOGS
-- ============================================================================

-- Audit logs table
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    old_data JSONB,
    new_data JSONB,
    changed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ip_address INET,
    user_agent TEXT
);

COMMENT ON TABLE audit_logs IS 'Audit trail for data changes';

-- ============================================================================
-- SECTION 12: INDEXES
-- ============================================================================

-- Users indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_cognito_user_id ON users(cognito_user_id);
CREATE INDEX idx_users_auth_provider ON users(auth_provider);

-- Social profiles indexes
CREATE INDEX idx_social_profiles_user_id ON social_profiles(user_id);
CREATE INDEX idx_social_profiles_provider ON social_profiles(provider, provider_id);

-- Sessions indexes
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Destinations indexes
CREATE INDEX idx_destinations_country ON destinations(country);
CREATE INDEX idx_destinations_port_code ON destinations(port_code) WHERE port_code IS NOT NULL;
CREATE INDEX idx_destinations_airport_code ON destinations(airport_code) WHERE airport_code IS NOT NULL;

-- Categories indexes
CREATE INDEX idx_categories_slug ON categories(slug);
CREATE INDEX idx_categories_parent_category_id ON categories(parent_category_id) WHERE parent_category_id IS NOT NULL;
CREATE INDEX idx_categories_active ON categories(active);

-- Suppliers indexes
CREATE INDEX idx_suppliers_active_status ON suppliers(active_status);
CREATE INDEX idx_suppliers_contract_type ON suppliers(contract_type);

-- Excursions indexes
CREATE INDEX idx_excursions_destination_id ON excursions(destination_id);
CREATE INDEX idx_excursions_supplier_id ON excursions(supplier_id);
CREATE INDEX idx_excursions_active_status ON excursions(active_status);
CREATE INDEX idx_excursions_difficulty_level ON excursions(difficulty_level);
CREATE INDEX idx_excursions_rating ON excursions(rating DESC);
CREATE INDEX idx_excursions_base_price ON excursions(base_price);

-- Excursion schedules indexes
CREATE INDEX idx_excursion_schedules_excursion_id ON excursion_schedules(excursion_id);

-- Cruise port calls indexes
CREATE INDEX idx_cruise_port_calls_destination_id ON cruise_port_calls(destination_id);
CREATE INDEX idx_cruise_port_calls_date ON cruise_port_calls(date);
CREATE INDEX idx_cruise_port_calls_cruise_line ON cruise_port_calls(cruise_line);
CREATE INDEX idx_cruise_port_calls_ship_name ON cruise_port_calls(ship_name);
CREATE INDEX idx_cruise_port_calls_status ON cruise_port_calls(status);

-- Flight arrivals indexes
CREATE INDEX idx_flight_arrivals_arrival_airport ON flight_arrivals(arrival_airport);
CREATE INDEX idx_flight_arrivals_date ON flight_arrivals(date);
CREATE INDEX idx_flight_arrivals_flight_number ON flight_arrivals(flight_number);
CREATE INDEX idx_flight_arrivals_status ON flight_arrivals(status);

-- Inventory records indexes
CREATE INDEX idx_inventory_records_excursion_id ON inventory_records(excursion_id);
CREATE INDEX idx_inventory_records_date ON inventory_records(date);
CREATE INDEX idx_inventory_records_excursion_date ON inventory_records(excursion_id, date);
CREATE INDEX idx_inventory_records_available_spots ON inventory_records(available_spots) WHERE available_spots > 0;

-- Pricing configurations indexes
CREATE INDEX idx_pricing_configurations_excursion_id ON pricing_configurations(excursion_id);

-- Bookings indexes
CREATE INDEX idx_bookings_user_id ON bookings(user_id);
CREATE INDEX idx_bookings_excursion_id ON bookings(excursion_id);
CREATE INDEX idx_bookings_date ON bookings(date);
CREATE INDEX idx_bookings_status ON bookings(status);
CREATE INDEX idx_bookings_payment_status ON bookings(payment_status);
CREATE INDEX idx_bookings_payment_intent_id ON bookings(payment_intent_id);
CREATE INDEX idx_bookings_cruise_port_call_id ON bookings(cruise_port_call_id) WHERE cruise_port_call_id IS NOT NULL;
CREATE INDEX idx_bookings_flight_arrival_id ON bookings(flight_arrival_id) WHERE flight_arrival_id IS NOT NULL;
CREATE INDEX idx_bookings_created_at ON bookings(created_at DESC);

-- Payments indexes
CREATE INDEX idx_payments_booking_id ON payments(booking_id);
CREATE INDEX idx_payments_payment_intent_id ON payments(payment_intent_id);
CREATE INDEX idx_payments_status ON payments(status);

-- Refunds indexes
CREATE INDEX idx_refunds_booking_id ON refunds(booking_id);
CREATE INDEX idx_refunds_payment_id ON refunds(payment_id);
CREATE INDEX idx_refunds_status ON refunds(status);

-- Reviews indexes
CREATE INDEX idx_reviews_excursion_id ON reviews(excursion_id);
CREATE INDEX idx_reviews_user_id ON reviews(user_id);
CREATE INDEX idx_reviews_booking_id ON reviews(booking_id);
CREATE INDEX idx_reviews_rating ON reviews(rating);
CREATE INDEX idx_reviews_created_at ON reviews(created_at DESC);

-- Audit logs indexes
CREATE INDEX idx_audit_logs_table_name ON audit_logs(table_name);
CREATE INDEX idx_audit_logs_record_id ON audit_logs(record_id);
CREATE INDEX idx_audit_logs_changed_at ON audit_logs(changed_at DESC);

-- ============================================================================
-- SECTION 13: TRIGGERS
-- ============================================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at trigger to all relevant tables
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_social_profiles_updated_at BEFORE UPDATE ON social_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_destinations_updated_at BEFORE UPDATE ON destinations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_categories_updated_at BEFORE UPDATE ON categories
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_suppliers_updated_at BEFORE UPDATE ON suppliers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_excursions_updated_at BEFORE UPDATE ON excursions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_excursion_schedules_updated_at BEFORE UPDATE ON excursion_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_cruise_port_calls_updated_at BEFORE UPDATE ON cruise_port_calls
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_flight_arrivals_updated_at BEFORE UPDATE ON flight_arrivals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pricing_configurations_updated_at BEFORE UPDATE ON pricing_configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_bookings_updated_at BEFORE UPDATE ON bookings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_payments_updated_at BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_refunds_updated_at BEFORE UPDATE ON refunds
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reviews_updated_at BEFORE UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update excursion rating from reviews
CREATE OR REPLACE FUNCTION update_excursion_rating()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE excursions
    SET rating = (
        SELECT COALESCE(AVG(rating), 0)
        FROM reviews
        WHERE excursion_id = COALESCE(NEW.excursion_id, OLD.excursion_id)
    ),
    review_count = (
        SELECT COUNT(*)
        FROM reviews
        WHERE excursion_id = COALESCE(NEW.excursion_id, OLD.excursion_id)
    )
    WHERE id = COALESCE(NEW.excursion_id, OLD.excursion_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Apply rating update trigger
CREATE TRIGGER update_excursion_rating_on_review_insert
    AFTER INSERT ON reviews
    FOR EACH ROW EXECUTE FUNCTION update_excursion_rating();

CREATE TRIGGER update_excursion_rating_on_review_update
    AFTER UPDATE ON reviews
    FOR EACH ROW EXECUTE FUNCTION update_excursion_rating();

CREATE TRIGGER update_excursion_rating_on_review_delete
    AFTER DELETE ON reviews
    FOR EACH ROW EXECUTE FUNCTION update_excursion_rating();

-- ============================================================================
-- SECTION 14: PG_CRON JOBS
-- ============================================================================

-- Schedule session cleanup job (runs every hour)
SELECT cron.schedule(
    'cleanup-expired-sessions',
    '0 * * * *',
    $$DELETE FROM sessions WHERE expires_at < NOW()$$
);

COMMENT ON EXTENSION pg_cron IS 'Scheduled job: cleanup-expired-sessions runs hourly to remove expired sessions';

-- ============================================================================
-- SECTION 15: SAMPLE DATA - CATEGORIES
-- ============================================================================

-- Insert sample categories
INSERT INTO categories (name, slug, description, display_order) VALUES
    ('Adventure', 'adventure', 'Thrilling outdoor activities and adventures', 1),
    ('Water Sports', 'water-sports', 'Ocean and water-based activities', 2),
    ('Cultural', 'cultural', 'Cultural experiences and historical explorers', 3),
    ('Nature & Wildlife', 'nature-wildlife', 'Nature explorers and wildlife encounters', 4),
    ('Food & Drink', 'food-drink', 'Culinary experiences and tastings', 5),
    ('Beach & Relaxation', 'beach-relaxation', 'Beach activities and relaxation', 6),
    ('Family Friendly', 'family-friendly', 'Activities suitable for all ages', 7),
    ('Extreme Sports', 'extreme-sports', 'High-adrenaline extreme activities', 8),
    ('Shopping', 'shopping', 'Shopping explorers and experiences', 9),
    ('Sightseeing', 'sightseeing', 'City explorers and sightseeing', 10)
ON CONFLICT (slug) DO NOTHING;

-- ============================================================================
-- SECTION 16: UTILITY FUNCTIONS
-- ============================================================================

-- Function to check booking eligibility based on cutoff time
CREATE OR REPLACE FUNCTION check_booking_cutoff(
    p_excursion_schedule_id UUID,
    p_booking_date DATE
)
RETURNS BOOLEAN AS $$
DECLARE
    v_cutoff_hours INTEGER;
    v_start_time TIME;
    v_departure_time TIMESTAMP WITH TIME ZONE;
    v_hours_until_departure NUMERIC;
BEGIN
    -- Get cutoff hours and start time
    SELECT cutoff_hours, start_time
    INTO v_cutoff_hours, v_start_time
    FROM excursion_schedules
    WHERE id = p_excursion_schedule_id;
    
    -- Calculate departure time
    v_departure_time := p_booking_date + v_start_time;
    
    -- Calculate hours until departure
    v_hours_until_departure := EXTRACT(EPOCH FROM (v_departure_time - NOW())) / 3600;
    
    -- Return TRUE if booking is allowed
    RETURN v_hours_until_departure >= v_cutoff_hours;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION check_booking_cutoff IS 'Validates if booking is within allowed cutoff time';

-- Function to calculate cruise time window
CREATE OR REPLACE FUNCTION calculate_cruise_time_window(
    p_cruise_port_call_id UUID
)
RETURNS TABLE(
    earliest_start TIMESTAMP WITH TIME ZONE,
    latest_end TIMESTAMP WITH TIME ZONE
) AS $$
DECLARE
    v_arrival_time TIMESTAMP WITH TIME ZONE;
    v_departure_time TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Get arrival and departure times
    SELECT arrival_time, departure_time
    INTO v_arrival_time, v_departure_time
    FROM cruise_port_calls
    WHERE id = p_cruise_port_call_id;
    
    -- Apply buffer rules: +1 hour after arrival, -1.5 hours before departure
    earliest_start := v_arrival_time + INTERVAL '1 hour';
    latest_end := v_departure_time - INTERVAL '90 minutes';
    
    RETURN QUERY SELECT earliest_start, latest_end;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_cruise_time_window IS 'Calculates eligible time window for cruise excursions';

-- Function to calculate flight earliest start time
CREATE OR REPLACE FUNCTION calculate_flight_earliest_start(
    p_flight_arrival_id UUID
)
RETURNS TIMESTAMP WITH TIME ZONE AS $$
DECLARE
    v_arrival_time TIMESTAMP WITH TIME ZONE;
    v_earliest_start TIMESTAMP WITH TIME ZONE;
BEGIN
    -- Get arrival time
    SELECT arrival_time
    INTO v_arrival_time
    FROM flight_arrivals
    WHERE id = p_flight_arrival_id;
    
    -- Apply transfer buffer: +2 hours after arrival
    v_earliest_start := v_arrival_time + INTERVAL '2 hours';
    
    RETURN v_earliest_start;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION calculate_flight_earliest_start IS 'Calculates earliest explorer start time after flight arrival';

-- Function to get available inventory
CREATE OR REPLACE FUNCTION get_available_inventory(
    p_excursion_id UUID,
    p_date DATE
)
RETURNS INTEGER AS $$
DECLARE
    v_available_spots INTEGER;
BEGIN
    SELECT available_spots
    INTO v_available_spots
    FROM inventory_records
    WHERE excursion_id = p_excursion_id
      AND date = p_date
      AND is_blackout = FALSE;
    
    RETURN COALESCE(v_available_spots, 0);
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION get_available_inventory IS 'Returns available spots for excursion on specific date';

-- ============================================================================
-- SECTION 17: VIEWS
-- ============================================================================

-- View for upcoming bookings with excursion details
CREATE OR REPLACE VIEW v_upcoming_bookings AS
SELECT 
    b.id AS booking_id,
    b.user_id,
    b.date AS booking_date,
    b.status,
    b.total_passengers,
    b.total_price,
    b.currency,
    b.voucher_url,
    e.id AS excursion_id,
    e.title AS excursion_title,
    e.meeting_point,
    es.start_time,
    es.end_time,
    d.name AS destination_name,
    d.timezone,
    s.company_name AS supplier_name,
    s.contact_phone AS supplier_phone,
    b.cruise_info,
    b.flight_info,
    b.emergency_contact
FROM bookings b
JOIN excursions e ON b.excursion_id = e.id
JOIN excursion_schedules es ON b.excursion_schedule_id = es.id
JOIN destinations d ON e.destination_id = d.id
JOIN suppliers s ON e.supplier_id = s.id
WHERE b.status = 'Confirmed'
  AND b.date >= CURRENT_DATE
ORDER BY b.date, es.start_time;

COMMENT ON VIEW v_upcoming_bookings IS 'View of confirmed upcoming bookings with full details';

-- View for excursion availability summary
CREATE OR REPLACE VIEW v_excursion_availability AS
SELECT 
    e.id AS excursion_id,
    e.title,
    e.destination_id,
    d.name AS destination_name,
    ir.date,
    ir.total_capacity,
    ir.booked_spots,
    ir.held_spots,
    ir.available_spots,
    ir.is_blackout,
    CASE 
        WHEN ir.is_blackout THEN 'Blackout'
        WHEN ir.available_spots = 0 THEN 'Sold Out'
        WHEN ir.available_spots <= 5 THEN 'Limited'
        ELSE 'Available'
    END AS availability_status
FROM excursions e
JOIN destinations d ON e.destination_id = d.id
LEFT JOIN inventory_records ir ON e.id = ir.excursion_id
WHERE e.active_status = TRUE
  AND (ir.date IS NULL OR ir.date >= CURRENT_DATE)
ORDER BY ir.date, e.title;

COMMENT ON VIEW v_excursion_availability IS 'Summary view of excursion availability by date';

-- View for supplier dashboard metrics
CREATE OR REPLACE VIEW v_supplier_dashboard AS
SELECT 
    s.id AS supplier_id,
    s.company_name,
    COUNT(DISTINCT e.id) AS active_excursions,
    COUNT(DISTINCT CASE WHEN b.date >= CURRENT_DATE AND b.status = 'Confirmed' THEN b.id END) AS upcoming_bookings,
    SUM(CASE WHEN b.date = CURRENT_DATE AND b.status = 'Confirmed' THEN b.total_price ELSE 0 END) AS today_revenue,
    SUM(CASE WHEN DATE_TRUNC('month', b.date) = DATE_TRUNC('month', CURRENT_DATE) AND b.status = 'Confirmed' THEN b.total_price ELSE 0 END) AS month_revenue,
    AVG(e.rating) AS average_rating,
    SUM(e.review_count) AS total_reviews
FROM suppliers s
LEFT JOIN excursions e ON s.id = e.supplier_id AND e.active_status = TRUE
LEFT JOIN bookings b ON e.id = b.excursion_id
WHERE s.active_status = TRUE
GROUP BY s.id, s.company_name;

COMMENT ON VIEW v_supplier_dashboard IS 'Dashboard metrics for suppliers';

-- View for revenue reporting
CREATE OR REPLACE VIEW v_revenue_report AS
SELECT 
    DATE_TRUNC('day', b.date) AS report_date,
    d.name AS destination_name,
    d.country,
    c.name AS category,
    COUNT(b.id) AS total_bookings,
    SUM(b.total_passengers) AS total_passengers,
    SUM(b.total_price) AS total_revenue,
    b.currency,
    AVG(b.total_price) AS average_booking_value
FROM bookings b
JOIN excursions e ON b.excursion_id = e.id
JOIN destinations d ON e.destination_id = d.id
CROSS JOIN LATERAL jsonb_array_elements_text(e.categories) AS cat(category_name)
LEFT JOIN categories c ON c.name = cat.category_name
WHERE b.status IN ('Confirmed', 'Completed')
GROUP BY DATE_TRUNC('day', b.date), d.name, d.country, c.name, b.currency
ORDER BY report_date DESC;

COMMENT ON VIEW v_revenue_report IS 'Revenue reporting by date, destination, and category';

-- ============================================================================
-- SECTION 18: SECURITY & PERMISSIONS
-- ============================================================================

-- Create read-only role for reporting
CREATE ROLE reporting_user WITH LOGIN PASSWORD 'change_me_in_production';
GRANT CONNECT ON DATABASE postgres TO reporting_user;
GRANT USAGE ON SCHEMA public TO reporting_user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO reporting_user;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO reporting_user;

-- Create application role with full access
CREATE ROLE app_user WITH LOGIN PASSWORD 'change_me_in_production';
GRANT CONNECT ON DATABASE postgres TO app_user;
GRANT USAGE ON SCHEMA public TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO app_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_user;

-- ============================================================================
-- SECTION 19: PERFORMANCE OPTIMIZATION
-- ============================================================================

-- Enable parallel query execution
ALTER DATABASE postgres SET max_parallel_workers_per_gather = 4;

-- Set work memory for complex queries
ALTER DATABASE postgres SET work_mem = '256MB';

-- Enable JIT compilation for complex queries
ALTER DATABASE postgres SET jit = on;

-- ============================================================================
-- SECTION 20: TRIP LIFECYCLE & RESTRICTION CONTROLS
-- ============================================================================
--
-- These tables drive all trip state enforcement across the service layer and UI.
-- The rules are loaded at runtime — no hardcoded state logic in application code.
--
-- Lifecycle (Option A — Strict Single-Use):
--   draft → active → shared → finalized → consumed
--   finalized and consumed are permanently immutable.
-- ============================================================================

-- ----------------------------------------------------------------------------
-- 20.1  trip_status_transitions
--       DB-driven FSM: defines every legal from → to status move.
--       The service layer queries this table before accepting any status change.
-- ----------------------------------------------------------------------------
CREATE TABLE trip_status_transitions (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_status     VARCHAR(50) NOT NULL,
    to_status       VARCHAR(50) NOT NULL,
    allowed_scopes  JSONB        NOT NULL DEFAULT '["owner"]',
    -- Which actor scopes may trigger this transition.
    -- Values: "owner", "shared", "system"
    requires_reason BOOLEAN      NOT NULL DEFAULT FALSE,
    description     TEXT,
    active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(from_status, to_status),
    CONSTRAINT valid_from_status CHECK (from_status IN ('draft','active','shared','finalized','consumed')),
    CONSTRAINT valid_to_status   CHECK (to_status   IN ('draft','active','shared','finalized','consumed'))
);

COMMENT ON TABLE trip_status_transitions IS
    'FSM table: every row is a legal trip status transition. '
    'The service layer validates against this before writing any status change. '
    'No transition rows = all mutations blocked.';

-- Seed the legal transitions (Option A — strict single-use, no reopen)
INSERT INTO trip_status_transitions (from_status, to_status, allowed_scopes, requires_reason, description) VALUES
    ('draft',     'active',    '["owner","system"]',         FALSE, 'Owner activates a draft trip'),
    ('active',    'shared',    '["owner"]',                  FALSE, 'Owner opens the trip for collaboration'),
    ('active',    'finalized', '["owner","system"]',         FALSE, 'Owner finalizes without sharing'),
    ('shared',    'active',    '["owner"]',                  FALSE, 'Owner pulls back from shared to active'),
    ('shared',    'finalized', '["owner","system"]',         FALSE, 'Owner finalizes a shared trip'),
    ('finalized', 'consumed',  '["system"]',                 FALSE, 'System marks trip consumed after booking'),
    -- Soft escape hatch: owner may archive any non-consumed trip with a reason
    ('draft',     'consumed',  '["owner"]',                  TRUE,  'Owner explicitly abandons a draft'),
    ('active',    'consumed',  '["owner"]',                  TRUE,  'Owner explicitly abandons an active trip'),
    ('shared',    'consumed',  '["owner"]',                  TRUE,  'Owner explicitly abandons a shared trip');

-- ----------------------------------------------------------------------------
-- 20.2  trip_operation_rules
--       Per-status rules that define what category of operation is permitted.
--       Loaded by the service layer to enforce read/write/share/book access.
-- ----------------------------------------------------------------------------
CREATE TABLE trip_operation_rules (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_status   VARCHAR(50)  NOT NULL,
    operation     VARCHAR(100) NOT NULL,
    -- Known operations: read, write_owner, write_contributor,
    --                   add_member, remove_member, submit_intake,
    --                   share, book, cancel, view_history
    allowed       BOOLEAN      NOT NULL DEFAULT TRUE,
    allowed_scopes JSONB       NOT NULL DEFAULT '["owner","shared","system"]',
    -- Scopes that may perform this operation when allowed=TRUE.
    reason        TEXT,
    active        BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(trip_status, operation),
    CONSTRAINT valid_op_status CHECK (trip_status IN ('draft','active','shared','finalized','consumed'))
);

COMMENT ON TABLE trip_operation_rules IS
    'Per-status permission table: the service layer enforces every incoming '
    'operation against the active row for the trip''s current status. '
    'allowed=FALSE rows act as explicit denies.';

INSERT INTO trip_operation_rules (trip_status, operation, allowed, allowed_scopes, reason) VALUES
    -- draft
    ('draft', 'read',              TRUE,  '["owner","system"]',              NULL),
    ('draft', 'write_owner',       TRUE,  '["owner"]',                       NULL),
    ('draft', 'write_contributor', FALSE, '[]',                              'Contributors may only write once trip is active or shared'),
    ('draft', 'add_member',        FALSE, '[]',                              'Members cannot be added until trip is active'),
    ('draft', 'remove_member',     FALSE, '[]',                              'No members to remove from a draft'),
    ('draft', 'submit_intake',     FALSE, '[]',                              'Intake submission requires active or shared status'),
    ('draft', 'share',             FALSE, '[]',                              'Trip must be active before sharing'),
    ('draft', 'book',              FALSE, '[]',                              'Booking requires finalized status'),
    ('draft', 'cancel',            FALSE, '[]',                              'No active booking to cancel from draft'),
    ('draft', 'view_history',      TRUE,  '["owner","system"]',              NULL),

    -- active
    ('active', 'read',              TRUE,  '["owner","system"]',              NULL),
    ('active', 'write_owner',       TRUE,  '["owner"]',                       NULL),
    ('active', 'write_contributor', FALSE, '[]',                              'Contributor writes require shared status'),
    ('active', 'add_member',        TRUE,  '["owner"]',                       NULL),
    ('active', 'remove_member',     TRUE,  '["owner"]',                       NULL),
    ('active', 'submit_intake',     TRUE,  '["owner"]',                       NULL),
    ('active', 'share',             TRUE,  '["owner"]',                       NULL),
    ('active', 'book',              FALSE, '[]',                              'Trip must be finalized before booking'),
    ('active', 'cancel',            FALSE, '[]',                              'Nothing to cancel from active trip'),
    ('active', 'view_history',      TRUE,  '["owner","system"]',              NULL),

    -- shared
    ('shared', 'read',              TRUE,  '["owner","shared","system"]',     NULL),
    ('shared', 'write_owner',       TRUE,  '["owner"]',                       NULL),
    ('shared', 'write_contributor', TRUE,  '["shared"]',                      NULL),
    ('shared', 'add_member',        TRUE,  '["owner"]',                       NULL),
    ('shared', 'remove_member',     TRUE,  '["owner"]',                       NULL),
    ('shared', 'submit_intake',     TRUE,  '["owner","shared"]',              NULL),
    ('shared', 'share',             FALSE, '[]',                              'Trip is already shared'),
    ('shared', 'book',              FALSE, '[]',                              'Trip must be finalized before booking'),
    ('shared', 'cancel',            FALSE, '[]',                              'Nothing to cancel from shared trip'),
    ('shared', 'view_history',      TRUE,  '["owner","shared","system"]',     NULL),

    -- finalized (immutable — read + book only)
    ('finalized', 'read',              TRUE,  '["owner","shared","system"]', NULL),
    ('finalized', 'write_owner',       FALSE, '[]',                          'Finalized trips are immutable'),
    ('finalized', 'write_contributor', FALSE, '[]',                          'Finalized trips are immutable'),
    ('finalized', 'add_member',        FALSE, '[]',                          'Finalized trips are immutable'),
    ('finalized', 'remove_member',     FALSE, '[]',                          'Finalized trips are immutable'),
    ('finalized', 'submit_intake',     FALSE, '[]',                          'Finalized trips are immutable'),
    ('finalized', 'share',             FALSE, '[]',                          'Finalized trips are immutable'),
    ('finalized', 'book',              TRUE,  '["owner","system"]',           NULL),
    ('finalized', 'cancel',            FALSE, '[]',                          'No booking to cancel from finalized trip'),
    ('finalized', 'view_history',      TRUE,  '["owner","shared","system"]', NULL),

    -- consumed (permanently read-only)
    ('consumed', 'read',              TRUE,  '["owner","shared","system"]', NULL),
    ('consumed', 'write_owner',       FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'write_contributor', FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'add_member',        FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'remove_member',     FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'submit_intake',     FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'share',             FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'book',              FALSE, '[]',                          'Consumed trips are permanently read-only'),
    ('consumed', 'cancel',            TRUE,  '["owner","system"]',          'Owner or system may cancel a booking on a consumed trip'),
    ('consumed', 'view_history',      TRUE,  '["owner","shared","system"]', NULL);

-- ----------------------------------------------------------------------------
-- 20.3  trip_ui_hints
--       Per-status UI directives loaded by the front-end.
--       Decouples presentation logic from hardcoded React conditions.
-- ----------------------------------------------------------------------------
CREATE TABLE trip_ui_hints (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_status       VARCHAR(50)  NOT NULL UNIQUE,
    banner_variant    VARCHAR(50)  NOT NULL DEFAULT 'info',
    -- Values: info, warning, success, error, readonly
    banner_message    TEXT         NOT NULL,
    show_edit_actions BOOLEAN      NOT NULL DEFAULT TRUE,
    show_new_trip_cta BOOLEAN      NOT NULL DEFAULT FALSE,
    cta_label         VARCHAR(100),
    cta_action        VARCHAR(100),
    -- Action key the UI dispatches on CTA click: e.g. 'start_new_trip'
    active            BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_ui_hint_status CHECK (trip_status IN ('draft','active','shared','finalized','consumed'))
);

COMMENT ON TABLE trip_ui_hints IS
    'UI directive rows loaded by the front-end to render status banners and '
    'control which actions are visible. Changes here are reflected in the UI '
    'without a code deployment.';

INSERT INTO trip_ui_hints
    (trip_status, banner_variant, banner_message, show_edit_actions, show_new_trip_cta, cta_label, cta_action)
VALUES
    ('draft',     'info',     'Your trip is in draft. Fill in your details and activate when ready.',
                               TRUE,  FALSE, NULL, NULL),
    ('active',    'info',     'Your trip is active. Complete all sections and finalize when ready.',
                               TRUE,  FALSE, NULL, NULL),
    ('shared',    'warning',  'This trip is open for collaboration. Finalize it to lock contributions.',
                               TRUE,  FALSE, NULL, NULL),
    ('finalized', 'success',  'This trip is finalized and ready to book. No further edits are allowed.',
                               FALSE, FALSE, NULL, NULL),
    ('consumed',  'readonly', 'This trip has been completed and is now read-only. Start a new trip to plan your next adventure.',
                               FALSE, TRUE,  'Start New Trip', 'start_new_trip');

-- ----------------------------------------------------------------------------
-- 20.4  trip_status_change_log
--       Immutable audit log of every trip status change.
-- ----------------------------------------------------------------------------
CREATE TABLE trip_status_change_log (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_plan_id    UUID         NOT NULL,
    -- NOTE: FK to trip_plans is in V002; omitted here to keep V001 self-contained.
    --       Add FK constraint via ALTER TABLE in V002 if execution order requires it.
    from_status     VARCHAR(50),
    to_status       VARCHAR(50)  NOT NULL,
    changed_by_user_id UUID,
    actor_scope     VARCHAR(50)  NOT NULL DEFAULT 'owner',
    reason          TEXT,
    source          VARCHAR(100) NOT NULL DEFAULT 'service',
    -- Values: service, system, admin_override
    correlation_id  VARCHAR(128),
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_log_from_status CHECK (from_status IS NULL OR from_status IN ('draft','active','shared','finalized','consumed')),
    CONSTRAINT valid_log_to_status   CHECK (to_status IN ('draft','active','shared','finalized','consumed')),
    CONSTRAINT valid_log_actor_scope CHECK (actor_scope IN ('owner','shared','system','admin'))
);

COMMENT ON TABLE trip_status_change_log IS
    'Append-only log of every trip lifecycle state change. '
    'Used for audit, debugging, and compliance reporting.';

-- Prevent updates and deletes on this log table
CREATE OR REPLACE FUNCTION prevent_trip_status_log_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'trip_status_change_log is immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_trip_status_log_update
    BEFORE UPDATE ON trip_status_change_log
    FOR EACH ROW EXECUTE FUNCTION prevent_trip_status_log_mutation();

CREATE TRIGGER prevent_trip_status_log_delete
    BEFORE DELETE ON trip_status_change_log
    FOR EACH ROW EXECUTE FUNCTION prevent_trip_status_log_mutation();

-- ----------------------------------------------------------------------------
-- 20.5  Indexes for runtime lookups
-- ----------------------------------------------------------------------------
CREATE INDEX idx_trip_status_transitions_from ON trip_status_transitions(from_status) WHERE active = TRUE;
CREATE INDEX idx_trip_operation_rules_status   ON trip_operation_rules(trip_status)   WHERE active = TRUE;
CREATE INDEX idx_trip_ui_hints_status          ON trip_ui_hints(trip_status)          WHERE active = TRUE;
CREATE INDEX idx_trip_status_change_log_trip   ON trip_status_change_log(trip_plan_id);
CREATE INDEX idx_trip_status_change_log_time   ON trip_status_change_log(created_at DESC);

-- ============================================================================
-- SECTION 21: MAINTENANCE
-- ============================================================================

-- Vacuum and analyze all tables
VACUUM ANALYZE;

-- ============================================================================
-- END OF SCHEMA
-- ============================================================================

-- Summary of schema objects created:
-- - 23 Tables (users, social_profiles, sessions, destinations, categories, suppliers,
--   excursions, excursion_schedules, cruise_port_calls, flight_arrivals,
--   inventory_records, pricing_configurations, bookings, payments, refunds,
--   reviews, audit_logs,
--   trip_status_transitions, trip_operation_rules, trip_ui_hints,
--   trip_status_change_log)
-- - 70+ Indexes for query optimization
-- - 17+ Triggers for automatic timestamp updates, rating calculations,
--   and immutability enforcement
-- - 1 pg_cron job for session cleanup
-- - 6 Utility functions for business logic
-- - 4 Views for reporting and dashboards
-- - 2 Database roles for security
-- - Sample category data + full trip lifecycle seed data

COMMENT ON DATABASE postgres IS 'Explorer Excursion Booking Platform - Production Ready Schema v1.1';
