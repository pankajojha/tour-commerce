-- =============================================================================
-- V004: System Prompt Versioning
-- =============================================================================
-- Stores all agent system prompts in the database with full version history.
-- Only one version per agent_name may be active at a time.
--
-- Agents:
--   hotel_booking_sanitizer   → hotel_agent.py sanitization prompt
--   flight_booking_sanitizer  → flight_agent.py sanitization prompt
--   cruise_booking_sanitizer  → cruise_agent.py sanitization prompt
--
-- Version lifecycle:
--   1. A new draft row is inserted (is_active = FALSE).
--   2. activate_system_prompt(id) atomically deactivates the previous
--      active version and activates the new one.
--   3. Old versions are retained for full audit history.
-- =============================================================================

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 1: Main table
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE system_prompts (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_name     VARCHAR(100) NOT NULL,
    version        INTEGER      NOT NULL,
    content        TEXT         NOT NULL,
    description    TEXT,            -- human-readable summary of what changed
    change_summary TEXT,            -- diff / changelog entry
    is_active      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_by     UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    activated_at   TIMESTAMP WITH TIME ZONE,
    deactivated_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT system_prompts_version_positive CHECK (version > 0),
    UNIQUE (agent_name, version)
);

-- Only one active version per agent at a time (partial unique index)
CREATE UNIQUE INDEX idx_system_prompts_one_active
    ON system_prompts(agent_name)
    WHERE is_active = TRUE;

CREATE INDEX idx_system_prompts_agent_name ON system_prompts(agent_name);
CREATE INDEX idx_system_prompts_created_at ON system_prompts(created_at DESC);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 2: Activation function
-- ─────────────────────────────────────────────────────────────────────────────
-- Atomically promotes a draft version to active, retiring the current active.
-- Returns the id of the newly activated row.

CREATE OR REPLACE FUNCTION activate_system_prompt(p_prompt_id UUID)
RETURNS UUID
LANGUAGE plpgsql
SECURITY DEFINER
AS $$
DECLARE
    v_agent_name VARCHAR(100);
BEGIN
    -- Resolve the agent_name of the target row
    SELECT agent_name INTO v_agent_name
    FROM system_prompts
    WHERE id = p_prompt_id;

    IF v_agent_name IS NULL THEN
        RAISE EXCEPTION 'system_prompt not found: %', p_prompt_id;
    END IF;

    -- Deactivate the current active version (if any)
    UPDATE system_prompts
    SET    is_active      = FALSE,
           deactivated_at = CURRENT_TIMESTAMP
    WHERE  agent_name = v_agent_name
      AND  is_active   = TRUE
      AND  id          <> p_prompt_id;

    -- Activate the requested version
    UPDATE system_prompts
    SET    is_active     = TRUE,
           activated_at  = CURRENT_TIMESTAMP,
           deactivated_at = NULL
    WHERE  id = p_prompt_id;

    RETURN p_prompt_id;
END;
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 3: Helper view — active prompts only
-- ─────────────────────────────────────────────────────────────────────────────

CREATE VIEW active_system_prompts AS
SELECT id, agent_name, version, content, description, activated_at
FROM   system_prompts
WHERE  is_active = TRUE;

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 4: RLS
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE system_prompts ENABLE ROW LEVEL SECURITY;

-- Any authenticated app user may read active prompts (agents need this at runtime)
CREATE POLICY system_prompts_read ON system_prompts
    FOR SELECT
    USING (
        is_active = TRUE
        OR user_has_permission(current_app_user_id(), 'policy:view', NULL)
    );

-- Only travel_manager and above (level ≤ 3) may insert / update
CREATE POLICY system_prompts_write ON system_prompts
    FOR ALL
    USING  (user_has_permission(current_app_user_id(), 'policy:edit',   NULL))
    WITH CHECK (user_has_permission(current_app_user_id(), 'policy:edit', NULL));

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 5: Audit trigger — log every activation change
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE system_prompt_activations (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_name      VARCHAR(100) NOT NULL,
    activated_prompt_id   UUID NOT NULL REFERENCES system_prompts(id),
    deactivated_prompt_id UUID         REFERENCES system_prompts(id),
    changed_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
    changed_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_spa_agent_name  ON system_prompt_activations(agent_name);
CREATE INDEX idx_spa_changed_at  ON system_prompt_activations(changed_at DESC);
