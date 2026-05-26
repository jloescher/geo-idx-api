-- +goose Up
-- Scope API tokens to domains; link staging hostnames to production parent.

ALTER TABLE domains
    ADD COLUMN parent_domain_id BIGINT NULL REFERENCES domains(id) ON DELETE CASCADE,
    ADD COLUMN is_staging BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE personal_access_tokens
    ADD COLUMN domain_id BIGINT NULL REFERENCES domains(id) ON DELETE CASCADE;

CREATE UNIQUE INDEX personal_access_tokens_domain_id_unique
    ON personal_access_tokens (domain_id)
    WHERE domain_id IS NOT NULL;

UPDATE domains
SET is_staging = TRUE
WHERE domain_slug LIKE 'staging.%';

UPDATE domains AS child
SET parent_domain_id = parent.id
FROM domains AS parent
WHERE child.is_staging = TRUE
  AND child.parent_domain_id IS NULL
  AND parent.is_staging = FALSE
  AND parent.parent_domain_id IS NULL
  AND child.user_id IS NOT DISTINCT FROM parent.user_id
  AND child.domain_slug = 'staging.' || parent.domain_slug;

-- Attach Production token when user has exactly one production root domain.
WITH prod_roots AS (
    SELECT user_id, MIN(id) AS prod_id, COUNT(*) AS n
    FROM domains
    WHERE is_staging = FALSE AND parent_domain_id IS NULL
    GROUP BY user_id
    HAVING COUNT(*) = 1
)
UPDATE personal_access_tokens AS t
SET domain_id = pr.prod_id
FROM prod_roots AS pr
WHERE t.tokenable_type = 'App\Models\User'
  AND t.tokenable_id = pr.user_id
  AND t.name = 'Production'
  AND t.domain_id IS NULL;

-- Attach Staging token when user has exactly one staging child linked to that prod.
WITH staging_only AS (
    SELECT d.user_id, MIN(d.id) AS staging_id, COUNT(*) AS n
    FROM domains AS d
    WHERE d.is_staging = TRUE AND d.parent_domain_id IS NOT NULL
    GROUP BY d.user_id
    HAVING COUNT(*) = 1
)
UPDATE personal_access_tokens AS t
SET domain_id = so.staging_id
FROM staging_only AS so
WHERE t.tokenable_type = 'App\Models\User'
  AND t.tokenable_id = so.user_id
  AND t.name = 'Staging'
  AND t.domain_id IS NULL;

-- +goose Down
DROP INDEX IF EXISTS personal_access_tokens_domain_id_unique;
ALTER TABLE personal_access_tokens DROP COLUMN IF EXISTS domain_id;
ALTER TABLE domains DROP COLUMN IF EXISTS is_staging;
ALTER TABLE domains DROP COLUMN IF EXISTS parent_domain_id;
