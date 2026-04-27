-- =============================================================================
-- V003: Corporate Policy Engine + Role-Based Access Control
-- =============================================================================
-- Roles (5-tier, highest to lowest):
--   platform_admin  → full control over everything
--   company_admin   → full control within their company
--   travel_manager  → policy authoring + override approvals + reports
--   dept_manager    → override approvals within their dept + view reports
--   traveler        → booking actions + view own policy limits
-- =============================================================================

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 1: Role hierarchy
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE roles (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(50) NOT NULL UNIQUE,
    level       INTEGER     NOT NULL UNIQUE,  -- 1 = highest, 5 = lowest
    description TEXT
);

INSERT INTO roles (name, level, description) VALUES
    ('platform_admin',  1, 'Full platform control — cross-company'),
    ('company_admin',   2, 'Full control within their company'),
    ('travel_manager',  3, 'Policy authoring, activation, overrides, reports'),
    ('dept_manager',    4, 'Override approvals for their department, view reports'),
    ('traveler',        5, 'Booking actions, view own policy limits only');

-- User role assignments (a user may have one role per company context)
CREATE TABLE user_roles (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_name      VARCHAR(50) NOT NULL REFERENCES roles(name) ON DELETE RESTRICT,
    company_id     UUID        REFERENCES users(id) ON DELETE CASCADE,  -- NULL for platform_admin
    department_id  UUID,                                                 -- scope for dept_manager
    assigned_by    UUID        REFERENCES users(id) ON DELETE SET NULL,
    assigned_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, company_id, role_name)
);

CREATE INDEX idx_user_roles_user_id    ON user_roles(user_id);
CREATE INDEX idx_user_roles_company_id ON user_roles(company_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 2: Operation permissions matrix
-- Each row grants an operation to a role (or all roles at or above a level).
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE role_permissions (
    id               SERIAL PRIMARY KEY,
    operation        VARCHAR(100) NOT NULL,   -- e.g. 'policy:create', 'override:approve'
    min_role_level   INTEGER      NOT NULL,   -- roles with level <= this value may perform it
    description      TEXT,
    UNIQUE (operation)
);

-- Policy operations
INSERT INTO role_permissions (operation, min_role_level, description) VALUES
    ('policy:view',     5, 'View policies and limits — all roles'),
    ('policy:create',   3, 'Create a new policy — travel_manager and above'),
    ('policy:edit',     3, 'Edit an existing policy — travel_manager and above'),
    ('policy:delete',   2, 'Delete a policy — company_admin and above'),
    ('policy:activate', 3, 'Activate / version a policy — travel_manager and above');

-- Override operations
INSERT INTO role_permissions (operation, min_role_level, description) VALUES
    ('override:view',    5, 'View own override requests — all roles'),
    ('override:request', 5, 'Request a policy override — all roles'),
    ('override:approve', 4, 'Approve or deny override requests — dept_manager and above'),
    ('override:deny',    4, 'Deny override requests — dept_manager and above');

-- Report operations
INSERT INTO role_permissions (operation, min_role_level, description) VALUES
    ('report:view',  4, 'View compliance reports and audit logs — dept_manager and above'),
    ('report:export',3, 'Export compliance data — travel_manager and above');

-- Role management operations
INSERT INTO role_permissions (operation, min_role_level, description) VALUES
    ('role:view',   4, 'View role assignments — dept_manager and above'),
    ('role:assign', 2, 'Assign roles to users — company_admin and above');

-- Booking operations
INSERT INTO role_permissions (operation, min_role_level, description) VALUES
    ('booking:view',   5, 'View bookings — all roles'),
    ('booking:create', 5, 'Create a booking — all roles'),
    ('booking:cancel', 5, 'Cancel own booking — all roles'),
    ('booking:cancel:any', 3, 'Cancel any booking — travel_manager and above');

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 3: Helper function — check if current app user has permission
-- ─────────────────────────────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION user_has_permission(
    p_user_id  UUID,
    p_operation VARCHAR(100),
    p_company_id UUID DEFAULT NULL
) RETURNS BOOLEAN
LANGUAGE sql
STABLE
SECURITY DEFINER
AS $$
    SELECT EXISTS (
        SELECT 1
        FROM user_roles ur
        JOIN roles r        ON r.name = ur.role_name
        JOIN role_permissions rp ON rp.operation = p_operation
        WHERE ur.user_id = p_user_id
          AND r.level <= rp.min_role_level
          AND (p_company_id IS NULL OR ur.company_id = p_company_id OR ur.company_id IS NULL)
    );
$$;

-- Convenience: return the effective role level for a user in a company context
CREATE OR REPLACE FUNCTION user_role_level(
    p_user_id    UUID,
    p_company_id UUID DEFAULT NULL
) RETURNS INTEGER
LANGUAGE sql
STABLE
SECURITY DEFINER
AS $$
    SELECT COALESCE(
        MIN(r.level),   -- return highest privilege (lowest level number)
        99              -- 99 = no role assigned
    )
    FROM user_roles ur
    JOIN roles r ON r.name = ur.role_name
    WHERE ur.user_id = p_user_id
      AND (p_company_id IS NULL OR ur.company_id = p_company_id OR ur.company_id IS NULL);
$$;

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 4: Corporate policy tables
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE company_policies (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    company_id      UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    scope           VARCHAR(50)  NOT NULL DEFAULT 'company'
                        CHECK (scope IN ('company', 'department', 'traveler_tier')),
    department_id   UUID,        -- set when scope = 'department'
    effective_from  DATE         NOT NULL,
    effective_until DATE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    version         INTEGER      NOT NULL DEFAULT 1,
    created_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
    updated_by      UUID         REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT policy_date_order CHECK (
        effective_until IS NULL OR effective_until >= effective_from
    )
);

CREATE INDEX idx_company_policies_company ON company_policies(company_id) WHERE is_active = TRUE;

-- Individual enforceable rules within a policy
CREATE TABLE policy_rules (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    policy_id        UUID        NOT NULL REFERENCES company_policies(id) ON DELETE CASCADE,
    category         VARCHAR(50) NOT NULL
                         CHECK (category IN ('hotel', 'flight', 'car', 'meal',
                                             'advance_booking', 'preferred_vendor')),
    rule_key         VARCHAR(100) NOT NULL,
    rule_value       JSONB        NOT NULL,
    -- Examples:
    --   hotel:    {"max_nightly_rate": 250, "currency": "USD",
    --              "city_tier_overrides": {"NYC": 350, "SF": 325}}
    --   flight:   {"max_cabin_class": "economy", "advance_booking_days": 14}
    --   car:      {"max_car_class": "standard",
    --              "preferred_vendors": ["Hertz", "Enterprise"]}
    --   meal:     {"daily_allowance": 75, "currency": "USD",
    --              "per_meal_caps": {"breakfast": 20, "lunch": 25, "dinner": 30}}
    enforcement_level VARCHAR(50) NOT NULL DEFAULT 'hard'
                          CHECK (enforcement_level IN ('hard', 'soft', 'advisory')),
    created_at        TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (policy_id, category, rule_key)
);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 5: Policy evaluation audit log
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE policy_evaluations (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id          UUID NOT NULL REFERENCES trip_plans(id) ON DELETE RESTRICT,
    user_id          UUID NOT NULL REFERENCES users(id)      ON DELETE RESTRICT,
    policy_id        UUID REFERENCES company_policies(id),
    category         VARCHAR(50) NOT NULL,
    input_snapshot   JSONB NOT NULL,
    result           VARCHAR(50) NOT NULL
                         CHECK (result IN ('compliant', 'non_compliant', 'advisory',
                                           'override_pending', 'override_approved')),
    violations       JSONB NOT NULL DEFAULT '[]',
    -- [{rule_key, rule_value, actual_value, message}]
    allowed_alternatives JSONB NOT NULL DEFAULT '[]',
    policy_version   INTEGER NOT NULL,
    correlation_id   VARCHAR(255),
    evaluated_at     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_policy_evals_trip    ON policy_evaluations(trip_id);
CREATE INDEX idx_policy_evals_user    ON policy_evaluations(user_id);
CREATE INDEX idx_policy_evals_result  ON policy_evaluations(result);
CREATE INDEX idx_policy_evals_evalauted_at ON policy_evaluations(evaluated_at);

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 6: Override requests
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE policy_overrides (
    id                 UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    evaluation_id      UUID NOT NULL REFERENCES policy_evaluations(id) ON DELETE RESTRICT,
    trip_id            UUID NOT NULL REFERENCES trip_plans(id)          ON DELETE RESTRICT,
    requester_user_id  UUID NOT NULL REFERENCES users(id)               ON DELETE RESTRICT,
    approver_user_id   UUID          REFERENCES users(id),
    reason_code        VARCHAR(100) NOT NULL,
    -- e.g. 'client_requirement', 'last_minute_booking', 'no_compliant_options'
    reason_note        TEXT,
    status             VARCHAR(50) NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'approved', 'denied', 'expired')),
    requested_at       TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    resolved_at        TIMESTAMP WITH TIME ZONE,
    sla_deadline       TIMESTAMP WITH TIME ZONE
                           GENERATED ALWAYS AS (requested_at + INTERVAL '48 hours') STORED
);

CREATE INDEX idx_overrides_trip       ON policy_overrides(trip_id);
CREATE INDEX idx_overrides_requester  ON policy_overrides(requester_user_id);
CREATE INDEX idx_overrides_approver   ON policy_overrides(approver_user_id);
CREATE INDEX idx_overrides_status     ON policy_overrides(status) WHERE status = 'pending';

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 7: Row-level security
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE company_policies    ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_rules        ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_evaluations  ENABLE ROW LEVEL SECURITY;
ALTER TABLE policy_overrides    ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_roles          ENABLE ROW LEVEL SECURITY;

-- company_policies: users may view policies for their company; only travel_manager+ may write
CREATE POLICY company_policies_select ON company_policies
    FOR SELECT
    USING (
        user_has_permission(current_app_user_id(), 'policy:view', company_id)
    );

CREATE POLICY company_policies_insert ON company_policies
    FOR INSERT
    WITH CHECK (
        user_has_permission(current_app_user_id(), 'policy:create', company_id)
    );

CREATE POLICY company_policies_update ON company_policies
    FOR UPDATE
    USING (
        user_has_permission(current_app_user_id(), 'policy:edit', company_id)
    )
    WITH CHECK (
        user_has_permission(current_app_user_id(), 'policy:edit', company_id)
    );

CREATE POLICY company_policies_delete ON company_policies
    FOR DELETE
    USING (
        user_has_permission(current_app_user_id(), 'policy:delete', company_id)
    );

-- policy_rules: inherit parent policy company_id for auth
CREATE POLICY policy_rules_select ON policy_rules
    FOR SELECT
    USING (
        EXISTS (
            SELECT 1 FROM company_policies cp
            WHERE cp.id = policy_rules.policy_id
              AND user_has_permission(current_app_user_id(), 'policy:view', cp.company_id)
        )
    );

CREATE POLICY policy_rules_write ON policy_rules
    FOR ALL
    USING (
        EXISTS (
            SELECT 1 FROM company_policies cp
            WHERE cp.id = policy_rules.policy_id
              AND user_has_permission(current_app_user_id(), 'policy:edit', cp.company_id)
        )
    )
    WITH CHECK (
        EXISTS (
            SELECT 1 FROM company_policies cp
            WHERE cp.id = policy_rules.policy_id
              AND user_has_permission(current_app_user_id(), 'policy:edit', cp.company_id)
        )
    );

-- policy_evaluations: travelers see only their own; managers see all in their company
CREATE POLICY policy_evals_select ON policy_evaluations
    FOR SELECT
    USING (
        user_id = current_app_user_id()
        OR EXISTS (
            SELECT 1 FROM company_policies cp
            WHERE cp.id = policy_evaluations.policy_id
              AND user_has_permission(current_app_user_id(), 'report:view', cp.company_id)
        )
    );

-- policy_evaluations: only the policy engine service account may insert
CREATE POLICY policy_evals_insert ON policy_evaluations
    FOR INSERT
    WITH CHECK (user_id = current_app_user_id());

-- policy_overrides: traveler sees own requests; dept_manager+ sees all pending in their scope
CREATE POLICY policy_overrides_select ON policy_overrides
    FOR SELECT
    USING (
        requester_user_id = current_app_user_id()
        OR user_has_permission(current_app_user_id(), 'override:approve')
    );

CREATE POLICY policy_overrides_insert ON policy_overrides
    FOR INSERT
    WITH CHECK (
        requester_user_id = current_app_user_id()
        AND user_has_permission(current_app_user_id(), 'override:request')
    );

CREATE POLICY policy_overrides_update ON policy_overrides
    FOR UPDATE
    USING (
        user_has_permission(current_app_user_id(), 'override:approve')
    )
    WITH CHECK (
        user_has_permission(current_app_user_id(), 'override:approve')
    );

-- user_roles: role:view permission to read; role:assign to write
CREATE POLICY user_roles_select ON user_roles
    FOR SELECT
    USING (
        user_id = current_app_user_id()
        OR user_has_permission(current_app_user_id(), 'role:view')
    );

CREATE POLICY user_roles_write ON user_roles
    FOR ALL
    USING (
        user_has_permission(current_app_user_id(), 'role:assign')
    )
    WITH CHECK (
        user_has_permission(current_app_user_id(), 'role:assign')
    );

-- ─────────────────────────────────────────────────────────────────────────────
-- SECTION 8: Cognito group → role sync table
-- Populated on login via JWT claims (Cognito groups map to role names 1:1)
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE cognito_role_sync (
    cognito_group  VARCHAR(100) NOT NULL,
    role_name      VARCHAR(50)  NOT NULL REFERENCES roles(name) ON DELETE CASCADE,
    PRIMARY KEY (cognito_group)
);

INSERT INTO cognito_role_sync (cognito_group, role_name) VALUES
    ('PlatformAdmins',  'platform_admin'),
    ('CompanyAdmins',   'company_admin'),
    ('TravelManagers',  'travel_manager'),
    ('DeptManagers',    'dept_manager'),
    ('Travelers',       'traveler');

COMMENT ON TABLE cognito_role_sync IS
    'Maps Cognito group names to internal role names. '
    'On login the Lambda authorizer resolves groups from JWT claims '
    'and ensures user_roles rows exist for the authenticated user.';
