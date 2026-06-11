-- 0001_init.down.sql — reverse of 0001_init.up.sql.

BEGIN;

DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS verdict_audit;
DROP TABLE IF EXISTS architectural_rulesets;
DROP TABLE IF EXISTS gold_codebase_index;
DROP TABLE IF EXISTS enterprise_organization;

-- The vector extension is left installed: other schemas may rely on it.

COMMIT;
