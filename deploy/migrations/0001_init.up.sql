-- 0001_init.up.sql — core schema for MEMBRANE.AI (ARCHITECTURE.md §4).
-- Requires the pgvector extension (bundled in the dev image pgvector/pgvector:pg16).

BEGIN;

CREATE EXTENSION IF NOT EXISTS vector;

-- Tenants.
CREATE TABLE IF NOT EXISTS enterprise_organization (
    org_id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name VARCHAR(255) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- "Gold codebase" reference index: the organization's best implementations,
-- embedded with OpenAI text-embedding-3-large (3072 dims) for RAG retrieval.
CREATE TABLE IF NOT EXISTS gold_codebase_index (
    vector_id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                UUID NOT NULL REFERENCES enterprise_organization(org_id) ON DELETE CASCADE,
    file_path             VARCHAR(512) NOT NULL,
    language_tag          VARCHAR(64)  NOT NULL,
    raw_code_content      TEXT         NOT NULL,
    architectural_context TEXT         NOT NULL,
    embedding             VECTOR(3072) NOT NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- HNSW caps plain vector columns at 2000 dims; for 3072-dim embeddings the
-- standard pattern is a halfvec (fp16) expression index. Queries must use the
-- same expression to hit it:  ORDER BY embedding::halfvec(3072) <=> $1::halfvec(3072).
CREATE INDEX IF NOT EXISTS gold_codebase_embedding_hnsw
    ON gold_codebase_index
    USING hnsw ((embedding::halfvec(3072)) halfvec_cosine_ops)
    WITH (m = 16, ef_construction = 64);

CREATE INDEX IF NOT EXISTS gold_codebase_org_lang
    ON gold_codebase_index (org_id, language_tag);

-- Per-organization architectural rules enforced by the pipeline.
CREATE TABLE IF NOT EXISTS architectural_rulesets (
    rule_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id     UUID NOT NULL REFERENCES enterprise_organization(org_id) ON DELETE CASCADE,
    rule       TEXT NOT NULL,
    severity   VARCHAR(32) NOT NULL CHECK (severity IN ('info', 'warning', 'blocking')),
    version    INTEGER     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS architectural_rulesets_org
    ON architectural_rulesets (org_id, version);

-- Tamper-evident verdict audit trail (append-only by convention). org_id is a
-- free-form tenant string (no FK): the audit log must never lose a verdict
-- because an org row is not provisioned yet — tenancy is enforced at the
-- gold-index boundary, not here (D-022).
CREATE TABLE IF NOT EXISTS verdict_audit (
    audit_id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    submission_id UUID         NOT NULL,
    org_id        VARCHAR(128) NOT NULL,
    verdict       VARCHAR(32)  NOT NULL,
    source        VARCHAR(32)  NOT NULL, -- cache | pipeline | fallback
    model         VARCHAR(128) NOT NULL DEFAULT '',
    detail        JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS verdict_audit_submission
    ON verdict_audit (submission_id);
CREATE INDEX IF NOT EXISTS verdict_audit_org_time
    ON verdict_audit (org_id, created_at);

-- Transactional outbox (D-013): domain row + outbox row are written in one
-- ACID transaction; a relay publishes rows where published_at IS NULL.
CREATE TABLE IF NOT EXISTS outbox (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    aggregate    VARCHAR(128) NOT NULL,
    topic        VARCHAR(128) NOT NULL,
    partition_key VARCHAR(128) NOT NULL DEFAULT '',
    payload      JSONB        NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS outbox_unpublished
    ON outbox (id)
    WHERE published_at IS NULL;

COMMIT;
