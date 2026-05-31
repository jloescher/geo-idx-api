-- Recommended: Create a dedicated read-only user for the MCP Monitor service.
-- This user should only have SELECT privileges on monitoring-related tables.

-- 1. Create the user (adjust password and host as needed for your environment)
-- For production Patroni, create the user on the primary and it will replicate.

CREATE USER mcp_monitor WITH PASSWORD 'change-me-strong-password';

-- 2. Grant connect and usage
GRANT CONNECT ON DATABASE geoidxapi TO mcp_monitor;
GRANT USAGE ON SCHEMA public TO mcp_monitor;

-- 3. Grant SELECT on the tables the MCP actually needs
-- Core monitoring tables
GRANT SELECT ON 
    domains,
    listings,
    jobs,
    failed_jobs,
    job_batches,
    replica_pages,
    listing_sync_cursors,
    gis_source_states,
    gis_parcels,
    gis_cities,
    gis_counties,
    gis_zips,
    gis_import_uploads,
    mls_proxy_audit_logs,
    sync_rate_budget,
    fema_enrichment_audit
TO mcp_monitor;

-- 4. (Optional but recommended) Create a read-only role and grant via the role
-- This makes future grants easier.

-- 5. In the MCP monitor config / environment, use a DSN with this user.
-- Example: postgres://mcp_monitor:xxx@primary-host:5432/geoidxapi?sslmode=require&application_name=idx-mcp-monitor

-- Important: Never give this user INSERT/UPDATE/DELETE or the ability to call functions that mutate state.
-- The MCP server is intentionally designed to be read-only.