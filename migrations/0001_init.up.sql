-- 0001_init.up.sql
-- Schema inicial do SaaS Multi-Tenant de Gestão de Chamados.
-- Estratégia de isolamento: coluna tenant_id em todas as tabelas de negócio (ADR-001).

PRAGMA foreign_keys = ON;

-- ============================================================
-- tenants — raiz do isolamento multi-tenant
-- ============================================================
CREATE TABLE tenants (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT     NOT NULL,
    slug       TEXT     NOT NULL UNIQUE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- users — pertence a exatamente um tenant
-- ============================================================
CREATE TABLE users (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id     INTEGER  NOT NULL REFERENCES tenants(id),
    name          TEXT     NOT NULL,
    email         TEXT     NOT NULL,
    password_hash TEXT     NOT NULL,
    role          TEXT     NOT NULL CHECK (role IN ('admin', 'agent', 'customer')),
    is_active     BOOLEAN  NOT NULL DEFAULT 1,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, email)
);

-- ============================================================
-- teams
-- ============================================================
CREATE TABLE teams (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id  INTEGER  NOT NULL REFERENCES tenants(id),
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

-- ============================================================
-- team_members — N:N entre users (agents) e teams
-- ============================================================
CREATE TABLE team_members (
    team_id   INTEGER NOT NULL REFERENCES teams(id),
    user_id   INTEGER NOT NULL REFERENCES users(id),
    tenant_id INTEGER NOT NULL REFERENCES tenants(id),
    PRIMARY KEY (team_id, user_id)
);

-- ============================================================
-- categories — definidas por cada tenant
-- ============================================================
CREATE TABLE categories (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id  INTEGER  NOT NULL REFERENCES tenants(id),
    name       TEXT     NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, name)
);

-- ============================================================
-- tickets — núcleo do produto
-- ============================================================
CREATE TABLE tickets (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id        INTEGER  NOT NULL REFERENCES tenants(id),
    title            TEXT     NOT NULL,
    description      TEXT     NOT NULL,
    status           TEXT     NOT NULL DEFAULT 'open'
                              CHECK (status IN ('open', 'in_progress', 'waiting_customer', 'resolved', 'closed')),
    priority         TEXT     NOT NULL DEFAULT 'medium'
                              CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    category_id      INTEGER  NOT NULL REFERENCES categories(id),
    created_by       INTEGER  NOT NULL REFERENCES users(id),
    assigned_to      INTEGER  REFERENCES users(id),
    assigned_team_id INTEGER  REFERENCES teams(id),
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at      DATETIME,
    closed_at        DATETIME
);

-- ============================================================
-- comments — público ou nota interna
-- ============================================================
CREATE TABLE comments (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id   INTEGER  NOT NULL REFERENCES tenants(id),
    ticket_id   INTEGER  NOT NULL REFERENCES tickets(id),
    author_id   INTEGER  NOT NULL REFERENCES users(id),
    body        TEXT     NOT NULL,
    is_internal BOOLEAN  NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- refresh_tokens — suporte ao fluxo de refresh JWT
-- ============================================================
CREATE TABLE refresh_tokens (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    tenant_id  INTEGER  NOT NULL REFERENCES tenants(id),
    user_id    INTEGER  NOT NULL REFERENCES users(id),
    token_hash TEXT     NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    revoked_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================
-- Índices (RNF08)
-- ============================================================
CREATE INDEX idx_users_tenant          ON users (tenant_id);
CREATE INDEX idx_teams_tenant          ON teams (tenant_id);
CREATE INDEX idx_categories_tenant     ON categories (tenant_id);
CREATE INDEX idx_tickets_tenant_status ON tickets (tenant_id, status);
CREATE INDEX idx_tickets_assigned      ON tickets (tenant_id, assigned_to);
CREATE INDEX idx_tickets_created_by    ON tickets (tenant_id, created_by);
CREATE INDEX idx_comments_ticket       ON comments (ticket_id, created_at);
CREATE INDEX idx_refresh_token_hash    ON refresh_tokens (token_hash);
