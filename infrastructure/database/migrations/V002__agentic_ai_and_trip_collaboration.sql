-- ============================================================================
-- V002: Agentic AI Orchestration + Trip Collaboration Controls
-- ============================================================================
-- Goals:
-- 1) Persist AI run metadata and immutable outputs with 3-year retention defaults
-- 2) Enforce webhook integrity/replay/idempotency tracking
-- 3) Add trip collaboration storage with ownership labels, soft delete, and version history
-- 4) Enable row-level security scaffolding for shared-trip access controls
--
-- NOTE:
-- - AgentCore does not access DB directly; service layer remains the only DB access path.
-- - Service layer must set app.current_user_id in-session before RLS-protected queries.
-- ============================================================================

-- ============================================================================
-- SECTION 1: APP CONTEXT HELPERS
-- ============================================================================

CREATE OR REPLACE FUNCTION current_app_user_id()
RETURNS UUID AS $$
DECLARE
    v_user_id TEXT;
BEGIN
    v_user_id := current_setting('app.current_user_id', true);
    IF v_user_id IS NULL OR length(trim(v_user_id)) = 0 THEN
        RETURN NULL;
    END IF;

    RETURN v_user_id::UUID;
EXCEPTION
    WHEN others THEN
        RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION current_app_user_id IS 'Returns UUID from session setting app.current_user_id for RLS policies';

-- ============================================================================
-- SECTION 2: TRIP COLLABORATION CORE TABLES
-- ============================================================================

CREATE TABLE trip_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    -- Status values must match trip_status_transitions / trip_operation_rules seed data in V001.
    -- 'consumed' is the only terminal state; the partial unique index below enforces one active trip per owner.
    status VARCHAR(50) NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'active', 'shared', 'finalized', 'consumed')),
    -- Trip window. trip_end_date is optional in draft but REQUIRED before the trip can leave draft.
    -- Validated by the enforce_trip_end_date_on_activation trigger below.
    trip_start_date DATE,
    trip_end_date   DATE,
    CONSTRAINT trip_date_order CHECK (
        trip_start_date IS NULL
        OR trip_end_date IS NULL
        OR trip_end_date >= trip_start_date
    ),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE trip_plans IS
    'Top-level trip planner entity with a single owning user. '
    'A user may only have one non-consumed trip at a time (enforced by uq_one_active_trip_per_owner). '
    'trip_end_date is mandatory before the trip can leave draft status.';

CREATE TABLE trip_plan_members (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_plan_id UUID NOT NULL REFERENCES trip_plans(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_scope VARCHAR(50) NOT NULL CHECK (member_scope IN ('owner', 'shared')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(trip_plan_id, user_id)
);

COMMENT ON TABLE trip_plan_members IS 'Membership/access map for trip plans';

CREATE TABLE trip_plan_sections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_plan_id UUID NOT NULL REFERENCES trip_plans(id) ON DELETE CASCADE,
    category VARCHAR(50) NOT NULL CHECK (category IN ('flight', 'cruise', 'hotel', 'car', 'excursions', 'budget', 'notes')),
    owner_override_allowed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(trip_plan_id, category)
);

COMMENT ON TABLE trip_plan_sections IS 'Per-trip section categories and owner override settings';
COMMENT ON COLUMN trip_plan_sections.owner_override_allowed IS 'Only true for owner-override categories such as budget/notes';

CREATE TABLE trip_plan_section_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    section_id UUID NOT NULL REFERENCES trip_plan_sections(id) ON DELETE CASCADE,
    contributor_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    ownership_label VARCHAR(255) NOT NULL,
    content JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    deleted_at TIMESTAMP WITH TIME ZONE,
    deleted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    delete_reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT valid_soft_delete_state CHECK (
        (is_deleted = FALSE AND deleted_at IS NULL AND deleted_by_user_id IS NULL)
        OR
        (is_deleted = TRUE AND deleted_at IS NOT NULL AND deleted_by_user_id IS NOT NULL)
    ),
    UNIQUE(section_id, contributor_user_id)
);

COMMENT ON TABLE trip_plan_section_entries IS 'Contributor-owned section content with soft-delete lifecycle';

CREATE TABLE trip_plan_entry_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entry_id UUID NOT NULL REFERENCES trip_plan_section_entries(id) ON DELETE RESTRICT,
    version_number INTEGER NOT NULL CHECK (version_number > 0),
    action VARCHAR(50) NOT NULL CHECK (action IN ('create', 'update', 'soft_delete', 'restore')),
    changed_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    reason TEXT,
    previous_value JSONB,
    new_value JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(entry_id, version_number)
);

COMMENT ON TABLE trip_plan_entry_versions IS 'Immutable version history for all entry create/update/delete/restore actions';

-- ============================================================================
-- SECTION 3: AGENTIC AI RUNS, IMMUTABLE OUTPUTS, TOKEN VAULT
-- ============================================================================

CREATE TABLE ai_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_plan_id UUID NOT NULL REFERENCES trip_plans(id) ON DELETE RESTRICT,
    requested_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    correlation_id VARCHAR(128) NOT NULL,
    run_id VARCHAR(128) NOT NULL,
    prompt_version VARCHAR(128) NOT NULL,
    model_version VARCHAR(128) NOT NULL,
    agent_version VARCHAR(128) NOT NULL,
    request_payload_redacted JSONB NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending' CHECK (status IN ('Pending', 'Running', 'Succeeded', 'Failed', 'Cancelled')),
    webhook_target TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(correlation_id, run_id)
);

COMMENT ON TABLE ai_runs IS 'Service-owned AI run registry with metadata for reproducibility and traceability';
COMMENT ON COLUMN ai_runs.request_payload_redacted IS 'Only redacted/tokenized payloads are allowed';

CREATE TABLE ai_run_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ai_run_id UUID NOT NULL REFERENCES ai_runs(id) ON DELETE CASCADE,
    correlation_id VARCHAR(128) NOT NULL,
    run_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('submitted', 'webhook_received', 'validated', 'duplicate', 'processed', 'failed')),
    event_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE ai_run_events IS 'Immutable event stream for run lifecycle and callback handling';

CREATE TABLE ai_outputs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ai_run_id UUID NOT NULL REFERENCES ai_runs(id) ON DELETE CASCADE,
    trip_plan_id UUID NOT NULL REFERENCES trip_plans(id) ON DELETE RESTRICT,
    correlation_id VARCHAR(128) NOT NULL,
    run_id VARCHAR(128) NOT NULL,
    output_type VARCHAR(100) NOT NULL,
    output_payload JSONB NOT NULL,
    prompt_version VARCHAR(128) NOT NULL,
    model_version VARCHAR(128) NOT NULL,
    agent_version VARCHAR(128) NOT NULL,
    legal_hold BOOLEAN NOT NULL DEFAULT FALSE,
    retain_until TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '3 years'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE ai_outputs IS 'Immutable AI outputs retained by default for 3 years unless legal_hold applies';

CREATE TABLE redaction_token_vault (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_plan_id UUID NOT NULL REFERENCES trip_plans(id) ON DELETE CASCADE,
    token VARCHAR(255) NOT NULL,
    token_type VARCHAR(50) NOT NULL CHECK (token_type IN ('pii', 'pci', 'sensitive')),
    original_value_encrypted TEXT NOT NULL,
    original_value_hash VARCHAR(128) NOT NULL,
    created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(trip_plan_id, token),
    UNIQUE(trip_plan_id, token_type, original_value_hash)
);

COMMENT ON TABLE redaction_token_vault IS 'Deterministic token vault for reversible sensitive value mappings';

-- ============================================================================
-- SECTION 4: WEBHOOK SECURITY, REPLAY, IDEMPOTENCY
-- ============================================================================

CREATE TABLE webhook_nonces (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source VARCHAR(100) NOT NULL,
    nonce VARCHAR(255) NOT NULL,
    signature_timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    correlation_id VARCHAR(128),
    run_id VARCHAR(128),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT (CURRENT_TIMESTAMP + INTERVAL '15 minutes'),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source, nonce)
);

COMMENT ON TABLE webhook_nonces IS 'Used nonce values for replay prevention; freshness window is 15 minutes';

CREATE TABLE webhook_callbacks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    source VARCHAR(100) NOT NULL,
    correlation_id VARCHAR(128) NOT NULL,
    run_id VARCHAR(128) NOT NULL,
    nonce VARCHAR(255),
    payload_hash VARCHAR(128),
    validation_status VARCHAR(50) NOT NULL CHECK (
        validation_status IN (
            'received',
            'rejected_signature',
            'rejected_timestamp',
            'rejected_nonce',
            'duplicate',
            'processed',
            'failed'
        )
    ),
    http_status INTEGER,
    response_code VARCHAR(100),
    response_message TEXT,
    received_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(source, correlation_id, run_id)
);

COMMENT ON TABLE webhook_callbacks IS 'Callback processing outcomes; duplicate correlation_id+run_id must signal duplicate';

-- ============================================================================
-- SECTION 5: INDEXES
-- ============================================================================

-- One active trip per owner: a user cannot start a new trip until the previous one is consumed.
-- The partial index covers all non-terminal statuses. INSERT will fail with a unique-violation
-- if the user already owns a draft/active/shared/finalized trip.
CREATE UNIQUE INDEX uq_one_active_trip_per_owner
    ON trip_plans(owner_user_id)
    WHERE status IN ('draft', 'active', 'shared', 'finalized');

CREATE INDEX idx_trip_plans_owner_user_id ON trip_plans(owner_user_id);
CREATE INDEX idx_trip_plan_members_trip_plan_id ON trip_plan_members(trip_plan_id);
CREATE INDEX idx_trip_plan_members_user_id ON trip_plan_members(user_id);
CREATE INDEX idx_trip_plan_sections_trip_plan_id ON trip_plan_sections(trip_plan_id);
CREATE INDEX idx_trip_plan_sections_category ON trip_plan_sections(category);
CREATE INDEX idx_trip_plan_section_entries_section_id ON trip_plan_section_entries(section_id);
CREATE INDEX idx_trip_plan_section_entries_contributor_user_id ON trip_plan_section_entries(contributor_user_id);
CREATE INDEX idx_trip_plan_section_entries_is_deleted ON trip_plan_section_entries(is_deleted);
CREATE INDEX idx_trip_plan_entry_versions_entry_id ON trip_plan_entry_versions(entry_id);
CREATE INDEX idx_trip_plan_entry_versions_created_at ON trip_plan_entry_versions(created_at DESC);

CREATE INDEX idx_ai_runs_trip_plan_id ON ai_runs(trip_plan_id);
CREATE INDEX idx_ai_runs_requested_by_user_id ON ai_runs(requested_by_user_id);
CREATE INDEX idx_ai_runs_correlation_run ON ai_runs(correlation_id, run_id);
CREATE INDEX idx_ai_runs_status ON ai_runs(status);
CREATE INDEX idx_ai_run_events_ai_run_id ON ai_run_events(ai_run_id);
CREATE INDEX idx_ai_outputs_ai_run_id ON ai_outputs(ai_run_id);
CREATE INDEX idx_ai_outputs_trip_plan_id ON ai_outputs(trip_plan_id);
CREATE INDEX idx_ai_outputs_retain_until ON ai_outputs(retain_until) WHERE legal_hold = FALSE;

CREATE INDEX idx_redaction_token_vault_trip_plan_id ON redaction_token_vault(trip_plan_id);
CREATE INDEX idx_webhook_nonces_expires_at ON webhook_nonces(expires_at);
CREATE INDEX idx_webhook_callbacks_validation_status ON webhook_callbacks(validation_status);
CREATE INDEX idx_webhook_callbacks_received_at ON webhook_callbacks(received_at DESC);

-- ============================================================================
-- SECTION 6: UPDATED_AT TRIGGERS FOR MUTABLE TABLES
-- ============================================================================

CREATE TRIGGER update_trip_plans_updated_at BEFORE UPDATE ON trip_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trip_plan_members_updated_at BEFORE UPDATE ON trip_plan_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trip_plan_sections_updated_at BEFORE UPDATE ON trip_plan_sections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_trip_plan_section_entries_updated_at BEFORE UPDATE ON trip_plan_section_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ai_runs_updated_at BEFORE UPDATE ON ai_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- SECTION 7: IMMUTABILITY + SOFT-DELETE ENFORCEMENT
-- ============================================================================

CREATE OR REPLACE FUNCTION prevent_hard_delete_trip_entry()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Hard delete is not allowed for trip_plan_section_entries. Use soft delete.';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_trip_entry_hard_delete
    BEFORE DELETE ON trip_plan_section_entries
    FOR EACH ROW EXECUTE FUNCTION prevent_hard_delete_trip_entry();

CREATE OR REPLACE FUNCTION prevent_mutation_trip_entry_versions()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'trip_plan_entry_versions is immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_trip_entry_versions_update
    BEFORE UPDATE ON trip_plan_entry_versions
    FOR EACH ROW EXECUTE FUNCTION prevent_mutation_trip_entry_versions();

CREATE TRIGGER prevent_trip_entry_versions_delete
    BEFORE DELETE ON trip_plan_entry_versions
    FOR EACH ROW EXECUTE FUNCTION prevent_mutation_trip_entry_versions();

CREATE OR REPLACE FUNCTION prevent_mutation_ai_run_events()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'ai_run_events is immutable';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_ai_run_events_update
    BEFORE UPDATE ON ai_run_events
    FOR EACH ROW EXECUTE FUNCTION prevent_mutation_ai_run_events();

CREATE TRIGGER prevent_ai_run_events_delete
    BEFORE DELETE ON ai_run_events
    FOR EACH ROW EXECUTE FUNCTION prevent_mutation_ai_run_events();

CREATE OR REPLACE FUNCTION prevent_mutation_ai_outputs()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'ai_outputs is immutable (insert-only).';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER prevent_ai_outputs_update
    BEFORE UPDATE ON ai_outputs
    FOR EACH ROW EXECUTE FUNCTION prevent_mutation_ai_outputs();

-- NOTE: DELETE on ai_outputs is intentionally allowed for retention cleanup once retain_until has passed.

-- ============================================================================
-- SECTION 7b: TRIP DATE ENFORCEMENT + ENTRY DATE-RANGE VALIDATION
-- ============================================================================

-- ── 7b.1  Enforce end-date before a trip leaves draft ──────────────────────
--
-- No trip may advance from 'draft' to any other status unless trip_end_date
-- is set. This is checked on EVERY status-changing UPDATE to trip_plans.
--
CREATE OR REPLACE FUNCTION enforce_trip_end_date_on_activation()
RETURNS TRIGGER AS $$
BEGIN
    -- Only act when status is actually changing away from 'draft'
    IF OLD.status = 'draft' AND NEW.status <> 'draft' THEN
        IF NEW.trip_end_date IS NULL THEN
            RAISE EXCEPTION
                'trip_end_date is required before a trip can leave draft status (trip_id: %)', NEW.id
                USING ERRCODE = 'check_violation';
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER enforce_trip_end_date_before_activation
    BEFORE UPDATE ON trip_plans
    FOR EACH ROW EXECUTE FUNCTION enforce_trip_end_date_on_activation();

-- ── 7b.2  Validate section entry dates are within the trip window ───────────
--
-- For time-bounded categories (flight, cruise, hotel, car), the content JSONB
-- MUST contain a "start_date" field (ISO-8601 DATE string). If the trip has
-- trip_end_date set, the entry start_date must be <= trip_end_date. If the trip
-- has trip_start_date set, the entry start_date must be >= trip_start_date.
--
-- Expected content schema per category:
--   flight   : {"departure_date": "YYYY-MM-DD", ...}
--   hotel    : {"check_in_date":  "YYYY-MM-DD", "check_out_date": "YYYY-MM-DD", ...}
--   cruise   : {"embark_date":    "YYYY-MM-DD", "disembark_date": "YYYY-MM-DD", ...}
--   car      : {"pickup_date":    "YYYY-MM-DD", "return_date":    "YYYY-MM-DD", ...}
--   excursions, budget, notes: not date-validated
--
CREATE OR REPLACE FUNCTION validate_section_entry_date_range()
RETURNS TRIGGER AS $$
DECLARE
    v_trip_start DATE;
    v_trip_end   DATE;
    v_category   VARCHAR(50);
    v_entry_start DATE;
    v_entry_end   DATE;
BEGIN
    -- Resolve the trip dates and category from the section
    SELECT p.trip_start_date, p.trip_end_date, s.category
    INTO v_trip_start, v_trip_end, v_category
    FROM trip_plan_sections s
    JOIN trip_plans p ON p.id = s.trip_plan_id
    WHERE s.id = NEW.section_id;

    -- Only validate time-bounded categories
    IF v_category NOT IN ('flight', 'hotel', 'cruise', 'car') THEN
        RETURN NEW;
    END IF;

    -- Extract entry start date by category
    v_entry_start := CASE v_category
        WHEN 'flight'  THEN (NEW.content ->> 'departure_date')::DATE
        WHEN 'hotel'   THEN (NEW.content ->> 'check_in_date')::DATE
        WHEN 'cruise'  THEN (NEW.content ->> 'embark_date')::DATE
        WHEN 'car'     THEN (NEW.content ->> 'pickup_date')::DATE
        ELSE NULL
    END;

    v_entry_end := CASE v_category
        WHEN 'flight'  THEN (NEW.content ->> 'arrival_date')::DATE
        WHEN 'hotel'   THEN (NEW.content ->> 'check_out_date')::DATE
        WHEN 'cruise'  THEN (NEW.content ->> 'disembark_date')::DATE
        WHEN 'car'     THEN (NEW.content ->> 'return_date')::DATE
        ELSE NULL
    END;

    -- If content has no date fields yet (still being filled), allow it
    IF v_entry_start IS NULL THEN
        RETURN NEW;
    END IF;

    -- Validate end precedes start within the entry itself
    IF v_entry_end IS NOT NULL AND v_entry_end < v_entry_start THEN
        RAISE EXCEPTION
            'Entry end date (%) is before start date (%) for category %',
            v_entry_end, v_entry_start, v_category
            USING ERRCODE = 'check_violation';
    END IF;

    -- Validate entry start is >= trip start (if trip start is set)
    IF v_trip_start IS NOT NULL AND v_entry_start < v_trip_start THEN
        RAISE EXCEPTION
            '% entry start date (%) is before the trip start date (%)',
            v_category, v_entry_start, v_trip_start
            USING ERRCODE = 'check_violation';
    END IF;

    -- Validate entry start is <= trip end (if trip end is set)
    IF v_trip_end IS NOT NULL AND v_entry_start > v_trip_end THEN
        RAISE EXCEPTION
            '% entry start date (%) is after the trip end date (%)',
            v_category, v_entry_start, v_trip_end
            USING ERRCODE = 'check_violation';
    END IF;

    -- Validate entry end (if present) is <= trip end
    IF v_trip_end IS NOT NULL AND v_entry_end IS NOT NULL AND v_entry_end > v_trip_end THEN
        RAISE EXCEPTION
            '% entry end date (%) extends beyond the trip end date (%)',
            v_category, v_entry_end, v_trip_end
            USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
EXCEPTION
    WHEN invalid_datetime_format OR invalid_text_representation THEN
        RAISE EXCEPTION
            'Invalid date format in % content for category %. Use YYYY-MM-DD.',
            v_category, v_category
            USING ERRCODE = 'invalid_datetime_format';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER validate_entry_dates_on_insert
    BEFORE INSERT ON trip_plan_section_entries
    FOR EACH ROW EXECUTE FUNCTION validate_section_entry_date_range();

CREATE TRIGGER validate_entry_dates_on_update
    BEFORE UPDATE OF content ON trip_plan_section_entries
    FOR EACH ROW EXECUTE FUNCTION validate_section_entry_date_range();

-- ============================================================================
-- SECTION 8: RLS ENABLEMENT + BASE POLICIES
-- ============================================================================

ALTER TABLE trip_plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE trip_plan_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE trip_plan_sections ENABLE ROW LEVEL SECURITY;
ALTER TABLE trip_plan_section_entries ENABLE ROW LEVEL SECURITY;

-- trip_plans: owner or member can read
CREATE POLICY trip_plans_select_policy ON trip_plans
FOR SELECT
USING (
    owner_user_id = current_app_user_id()
    OR EXISTS (
        SELECT 1
        FROM trip_plan_members m
        WHERE m.trip_plan_id = trip_plans.id
          AND m.user_id = current_app_user_id()
    )
);

-- trip_plans: owner manages
CREATE POLICY trip_plans_update_policy ON trip_plans
FOR UPDATE
USING (owner_user_id = current_app_user_id())
WITH CHECK (owner_user_id = current_app_user_id());

-- members: visible to trip owner and the member record owner
CREATE POLICY trip_plan_members_select_policy ON trip_plan_members
FOR SELECT
USING (
    user_id = current_app_user_id()
    OR EXISTS (
        SELECT 1
        FROM trip_plans p
        WHERE p.id = trip_plan_members.trip_plan_id
          AND p.owner_user_id = current_app_user_id()
    )
);

-- sections: visible to owner or members
CREATE POLICY trip_plan_sections_select_policy ON trip_plan_sections
FOR SELECT
USING (
    EXISTS (
        SELECT 1
        FROM trip_plans p
        WHERE p.id = trip_plan_sections.trip_plan_id
          AND (
              p.owner_user_id = current_app_user_id()
              OR EXISTS (
                  SELECT 1 FROM trip_plan_members m
                  WHERE m.trip_plan_id = p.id
                    AND m.user_id = current_app_user_id()
              )
          )
    )
);

-- section entries: visible to owner or members
CREATE POLICY trip_plan_section_entries_select_policy ON trip_plan_section_entries
FOR SELECT
USING (
    EXISTS (
        SELECT 1
        FROM trip_plan_sections s
        JOIN trip_plans p ON p.id = s.trip_plan_id
        WHERE s.id = trip_plan_section_entries.section_id
          AND (
              p.owner_user_id = current_app_user_id()
              OR EXISTS (
                  SELECT 1 FROM trip_plan_members m
                  WHERE m.trip_plan_id = p.id
                    AND m.user_id = current_app_user_id()
              )
          )
    )
);

-- insert: contributor can create their own row; only allowed when trip status
-- permits write_owner (if caller is trip owner) or write_contributor (if shared member).
-- Checked by the trip_entry_write_allowed() helper function.
CREATE OR REPLACE FUNCTION trip_entry_write_allowed(p_section_id UUID, p_caller_id UUID)
RETURNS BOOLEAN AS $$
DECLARE
    v_trip_status    VARCHAR(50);
    v_is_owner       BOOLEAN;
    v_operation      VARCHAR(100);
    v_allowed        BOOLEAN;
    v_scopes_raw     JSONB;
BEGIN
    -- Resolve trip status and whether caller is the owner
    SELECT p.status, (p.owner_user_id = p_caller_id)
    INTO v_trip_status, v_is_owner
    FROM trip_plan_sections s
    JOIN trip_plans p ON p.id = s.trip_plan_id
    WHERE s.id = p_section_id;

    IF NOT FOUND THEN
        RETURN FALSE;
    END IF;

    -- Owner uses write_owner; shared member uses write_contributor
    v_operation := CASE WHEN v_is_owner THEN 'write_owner' ELSE 'write_contributor' END;

    SELECT r.allowed, r.allowed_scopes
    INTO v_allowed, v_scopes_raw
    FROM trip_operation_rules r
    WHERE r.trip_status = v_trip_status
      AND r.operation   = v_operation
      AND r.active      = TRUE;

    IF NOT FOUND OR NOT v_allowed THEN
        RETURN FALSE;
    END IF;

    -- Check that the caller's scope appears in allowed_scopes
    RETURN CASE
        WHEN v_is_owner    THEN v_scopes_raw ? 'owner'
        WHEN EXISTS (
            SELECT 1 FROM trip_plan_members m
            JOIN trip_plan_sections s ON s.trip_plan_id = m.trip_plan_id
            WHERE s.id = p_section_id AND m.user_id = p_caller_id
        ) THEN v_scopes_raw ? 'shared'
        ELSE FALSE
    END;
END;
$$ LANGUAGE plpgsql STABLE;

COMMENT ON FUNCTION trip_entry_write_allowed IS
    'Returns TRUE when the caller may write a section entry based on current trip status '
    'and the trip_operation_rules table. The owner can write their own entries in shared mode. '
    'Nobody may write in finalized or consumed status.';

CREATE POLICY trip_plan_section_entries_insert_policy ON trip_plan_section_entries
FOR INSERT
WITH CHECK (
    -- Caller must own the entry row being inserted
    contributor_user_id = current_app_user_id()
    -- Trip status must allow the write for the caller's role
    AND trip_entry_write_allowed(section_id, current_app_user_id())
);

-- update: same rules — contributor edits only their own entry, and only when
-- the trip status allows writing (owner in shared mode CAN edit their own rows).
CREATE POLICY trip_plan_section_entries_update_policy ON trip_plan_section_entries
FOR UPDATE
USING (
    contributor_user_id = current_app_user_id()
    AND trip_entry_write_allowed(section_id, current_app_user_id())
)
WITH CHECK (
    contributor_user_id = current_app_user_id()
    AND trip_entry_write_allowed(section_id, current_app_user_id())
);

-- ============================================================================
-- SECTION 9: OPERATIONAL PG_CRON JOBS
-- ============================================================================

-- cleanup expired webhook nonces every hour
SELECT cron.schedule(
    'cleanup-expired-webhook-nonces',
    '0 * * * *',
    $$DELETE FROM webhook_nonces WHERE expires_at < NOW()$$
);

-- cleanup AI outputs past retention (excluding legal_hold) daily at 03:00
SELECT cron.schedule(
    'cleanup-expired-ai-outputs',
    '0 3 * * *',
    $$DELETE FROM ai_outputs WHERE legal_hold = FALSE AND retain_until < NOW()$$
);

-- ============================================================================
-- END OF V002
-- ============================================================================
