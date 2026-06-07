-- ================================================================
--  Smart Alloy Selector — PostgreSQL Schema
--  MET-QUEST '26 | Neon Serverless PostgreSQL
-- ================================================================

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;   -- for fuzzy name search
CREATE EXTENSION IF NOT EXISTS unaccent;  -- for accent-insensitive search
CREATE EXTENSION IF NOT EXISTS vector;    -- for semantic vector retrieval

-- ----------------------------------------------------------------
--  Main materials table
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS materials (
    id                       SERIAL PRIMARY KEY,
    name                     TEXT NOT NULL,
    formula                  TEXT,
    category                 TEXT,              -- Metal | Ceramic | Polymer | Composite | Semiconductor
    subcategory              TEXT,              -- e.g. Ferrous, Non-Ferrous, Oxide Ceramic …

    -- Core physical properties
    density                  FLOAT,             -- g/cm³
    glass_transition_temp    FLOAT,             -- Kelvin (polymers)
    heat_deflection_temp     FLOAT,             -- Kelvin (polymers)
    melting_point            FLOAT,             -- Kelvin
    boiling_point            FLOAT,             -- Kelvin (if available)
    thermal_conductivity     FLOAT,             -- W/(m·K)
    specific_heat            FLOAT,             -- J/(kg·K)
    thermal_expansion        FLOAT,             -- 10⁻⁶ /K (CTE)

    -- Electrical properties
    electrical_resistivity   FLOAT,             -- Ω·m (×10⁻⁸)

    -- Mechanical properties
    yield_strength           FLOAT,             -- MPa
    tensile_strength         FLOAT,             -- MPa
    youngs_modulus           FLOAT,             -- GPa
    hardness_vickers         FLOAT,             -- HV
    poissons_ratio           FLOAT,
    processing_temp_min_c    FLOAT,
    processing_temp_max_c    FLOAT,
    crystallinity            FLOAT,
    crystal_system           TEXT,
    fracture_toughness       FLOAT,
    weibull_modulus          FLOAT,
    interlaminar_shear_strength FLOAT,
    fiber_volume_fraction    FLOAT,

    -- Metadata
    source                   TEXT DEFAULT 'Materials Project',
    mp_material_id           TEXT UNIQUE,       -- e.g. "mp-66"
    notes                    TEXT,
    created_at               TIMESTAMPTZ DEFAULT NOW()
);

-- ----------------------------------------------------------------
--  Indexes for fast range queries (the RAG retrieval layer)
-- ----------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_mat_density         ON materials(density);
CREATE INDEX IF NOT EXISTS idx_mat_tg              ON materials(glass_transition_temp);
CREATE INDEX IF NOT EXISTS idx_mat_hdt             ON materials(heat_deflection_temp);
CREATE INDEX IF NOT EXISTS idx_mat_melting_pt       ON materials(melting_point);
CREATE INDEX IF NOT EXISTS idx_mat_thermal_cond     ON materials(thermal_conductivity);
CREATE INDEX IF NOT EXISTS idx_mat_resistivity      ON materials(electrical_resistivity);
CREATE INDEX IF NOT EXISTS idx_mat_yield_strength   ON materials(yield_strength);
CREATE INDEX IF NOT EXISTS idx_mat_youngs_modulus   ON materials(youngs_modulus);
CREATE INDEX IF NOT EXISTS idx_mat_category         ON materials(category);
CREATE INDEX IF NOT EXISTS idx_mat_formula          ON materials(formula);

-- Full-text / trigram index for name search
CREATE INDEX IF NOT EXISTS idx_mat_name_trgm 
    ON materials USING GIN (name gin_trgm_ops);

-- Trigram indexes for other text fields queried with ILIKE
CREATE INDEX IF NOT EXISTS idx_mat_category_trgm 
    ON materials USING GIN (category gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_mat_formula_trgm 
    ON materials USING GIN (formula gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_mat_subcategory_trgm 
    ON materials USING GIN (subcategory gin_trgm_ops);

-- Backward-compatible migration for existing DBs
ALTER TABLE materials ADD COLUMN IF NOT EXISTS glass_transition_temp FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS heat_deflection_temp FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS processing_temp_min_c FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS processing_temp_max_c FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS crystallinity FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS crystal_system TEXT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS fracture_toughness FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS weibull_modulus FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS interlaminar_shear_strength FLOAT;
ALTER TABLE materials ADD COLUMN IF NOT EXISTS fiber_volume_fraction FLOAT;

-- ----------------------------------------------------------------
--  Query log — track what users ask (useful for evaluation)
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS query_log (
    id              SERIAL PRIMARY KEY,
    raw_query       TEXT NOT NULL,
    extracted_json  JSONB,
    result_ids      INT[],
    created_at      TIMESTAMPTZ DEFAULT NOW()
);

-- ----------------------------------------------------------------
--  Material embeddings (semantic vector retrieval)
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS material_embeddings (
    material_id      INT PRIMARY KEY REFERENCES materials(id) ON DELETE CASCADE,
    embedding        vector(768),
    embedding_model  TEXT NOT NULL DEFAULT 'text-embedding-004',
    updated_at       TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mat_embeddings_hnsw
    ON material_embeddings USING hnsw (embedding vector_cosine_ops);

-- ----------------------------------------------------------------
--  Users, Chats and Messages (conversation persistence)
-- ----------------------------------------------------------------
-- Users table stores authenticated users. Use BIGSERIAL for easy indexing
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    full_name     TEXT,
    avatar_url    TEXT,
    provider      TEXT, -- e.g. google
    provider_id   TEXT, -- id from provider
    metadata      JSONB,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

CREATE TABLE IF NOT EXISTS chats (
    id            BIGSERIAL PRIMARY KEY,
    -- store user identifiers as TEXT to remain compatible with existing user tables
    user_id       TEXT,
    title         TEXT,
    state         JSONB,      -- e.g. routing info, last run context
    is_active     BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chats_user_id ON chats(user_id);

-- Messages are the atomic chat turns. Content is stored as JSONB
CREATE TABLE IF NOT EXISTS messages (
    id            BIGSERIAL PRIMARY KEY,
    -- keep chat_id as BIGINT (no FK) so creation stays safe across DBs
    chat_id       BIGINT,
    sender_role   TEXT NOT NULL, -- 'user' | 'assistant' | 'system'
    sender_id     TEXT,        -- optional link to users.id when sender is a user
    content       JSONB NOT NULL, -- structured payload (report, metadata, citations)
    content_text  TEXT,          -- simple text fallback for UI rendering / search
    tokens_used   INT DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Ensure columns exist on preexisting `messages` tables before creating indexes
ALTER TABLE messages ADD COLUMN IF NOT EXISTS chat_id BIGINT;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_id TEXT;

CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);

-- Backfill-safe upgrades (idempotent)
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE chats ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE;
ALTER TABLE messages ADD COLUMN IF NOT EXISTS content_text TEXT;
