package search

// Mirror window invariants (Active/Pending PostGIS leg):
//
//  1. Replication bulk-loads Active + Pending only (never Closed).
//  2. DecideRoute sends Active/Pending to PostGIS; Closed and mixed status to live upstream.
//  3. MLS_LOCAL_MIRROR_ROLLING_MONTHS + daily mls.purge_closed_listings trim stale rows.
//  4. Bridge Stellar replication seed uses status filter only (no timestamp $filter on /replication).
//
// Closed inventory is fetched on demand via LiveSearch (RESO OData, Web API fallback on 404).
