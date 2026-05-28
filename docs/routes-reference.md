# Route Reference (Code-Aligned)

This reference is derived from route registration in:

- `internal/api/routes.go`
- `internal/handler/dashboard/handler.go` (`Register`)
- `internal/api/static.go`

## JSON/API endpoints (Yaak/OpenAPI scope)

These endpoints are represented in `docs/yaak-api-collection.json`.

| Method | Path | Auth | Handler | Purpose |
|---|---|---|---|---|
| GET | `/healthz` | None | `healthz` | Liveness check (`{"status":"ok"}`). |
| GET | `/readyz` | None | `readyz` | Readiness check (DB ping + PostGIS version). |
| GET | `/health/replicas` | None | `healthReplicas` | Read-replica selector health snapshot. |
| GET | `/metrics` | None | Prometheus handler | Prometheus metrics scrape endpoint. |
| GET | `/openapi.json` | None | `openapi.serveSpec` | OpenAPI 3.1 spec (embedded from `docs/yaak-api-collection.json`). |
| GET | `/swagger` | None | `openapi.serveUI` | Swagger UI (loads `/openapi.json`). |
| POST | `/api/auth/token` | None | `auth.Handler.Token` | Email/password login for PAT issuance (`idx:full`). |
| GET | `/api/auth/user` | Bearer PAT | `auth.Handler.User` | Return current user profile from bearer token. |
| GET | `/images/:listingKey/:photoId` | `DomainToken` | `images.Handler.Show` | Stream listing photo via proxy/cache. |
| GET | `/api/v1/gis` | `DomainToken` + `MLSAccess` (GIS bypass in access middleware) | `gis.Handler.Show` | GIS parcel GeoJSON by bbox. |
| GET | `/api/v1/mls/:mlsCode/gis` | `DomainToken` + `MLSAccess` (GIS bypass in access middleware) | `gis.Handler.ShowForMLS` | MLS-scoped GIS parcel GeoJSON. |
| GET | `/api/v1/gis/autocomplete/cities` | `DomainToken` + `MLSAccess` | `gis.Handler.AutocompleteCities` | City \| county suggestions (`q`, optional `limit`). Requires `pg_trgm` (migration 00007). |
| GET | `/api/v1/gis/autocomplete/counties` | `DomainToken` + `MLSAccess` | `gis.Handler.AutocompleteCounties` | County name/slug suggestions (`q`, optional `limit`). |
| GET | `/api/v1/listings` | `DomainToken` + `MLSAccess` | `bridge.Handler.Listings` | MLS web listings collection proxy. |
| GET | `/api/v1/listings/:listingId` | `DomainToken` + `MLSAccess` | `bridge.Handler.Listing` | MLS listing detail proxy. |
| GET | `/api/v1/agents` | `DomainToken` + `MLSAccess` | `bridge.Handler.Agents` | MLS agents collection proxy. |
| GET | `/api/v1/agents/:agentId` | `DomainToken` + `MLSAccess` | `bridge.Handler.Agent` | MLS agent detail proxy. |
| GET | `/api/v1/offices` | `DomainToken` + `MLSAccess` | `bridge.Handler.Offices` | MLS offices collection proxy. |
| GET | `/api/v1/offices/:officeId` | `DomainToken` + `MLSAccess` | `bridge.Handler.Office` | MLS office detail proxy. |
| GET | `/api/v1/openhouses` | `DomainToken` + `MLSAccess` | `bridge.Handler.OpenHouses` | MLS open houses collection proxy. |
| GET | `/api/v1/openhouses/:openhouseId` | `DomainToken` + `MLSAccess` | `bridge.Handler.OpenHouse` | MLS open house detail proxy. |
| GET | `/api/v1/properties` | `DomainToken` + `MLSAccess` | `bridge.Handler.Properties` | RESO `Property` collection proxy. |
| POST | `/api/v1/properties` | `DomainToken` + `MLSAccess` | `bridge.Handler.Properties` | JSON convenience entry for property search/proxy. |
| GET | `/api/v1/properties/:listingKey` | `DomainToken` + `MLSAccess` | `bridge.Handler.Property` | RESO `Property('<key>')` entity proxy. |
| GET | `/api/v1/members` | `DomainToken` + `MLSAccess` | `bridge.Handler.Members` | RESO `Member` collection proxy. |
| GET | `/api/v1/members/:memberKey` | `DomainToken` + `MLSAccess` | `bridge.Handler.Member` | RESO `Member('<key>')` entity proxy. |
| GET | `/api/v1/reso-offices` | `DomainToken` + `MLSAccess` | `bridge.Handler.ResoOffices` | RESO `Office` collection proxy. |
| GET | `/api/v1/reso-offices/:officeKey` | `DomainToken` + `MLSAccess` | `bridge.Handler.ResoOffice` | RESO `Office('<key>')` entity proxy. |
| GET | `/api/v1/reso-openhouses` | `DomainToken` + `MLSAccess` | `bridge.Handler.ResoOpenHouses` | RESO `OpenHouse` collection proxy. |
| GET | `/api/v1/reso-openhouses/:openHouseKey` | `DomainToken` + `MLSAccess` | `bridge.Handler.ResoOpenHouse` | RESO `OpenHouse('<key>')` entity proxy. |
| GET | `/api/v1/lookup` | `DomainToken` + `MLSAccess` | `bridge.Handler.Lookup` | RESO lookup proxy (lookup cache partition). |
| GET | `/api/v1/pub/parcels` | `DomainToken` + `MLSAccess` | `bridge.Handler.PubParcels` | Public parcel collection proxy. |
| GET | `/api/v1/pub/parcels/:parcelId` | `DomainToken` + `MLSAccess` | `bridge.Handler.PubParcel` | Public parcel detail proxy. |
| GET | `/api/v1/pub/parcels/:parcelId/assessments` | `DomainToken` + `MLSAccess` | `bridge.Handler.PubParcelAssessments` | Public parcel assessments proxy. |
| GET | `/api/v1/pub/parcels/:parcelId/transactions` | `DomainToken` + `MLSAccess` | `bridge.Handler.PubParcelTransactions` | Public parcel transactions proxy. |
| GET | `/api/v1/pub/assessments` | `DomainToken` + `MLSAccess` | `bridge.Handler.PubAssessments` | Public assessments collection proxy. |
| GET | `/api/v1/pub/transactions` | `DomainToken` + `MLSAccess` | `bridge.Handler.PubTransactions` | Public transactions collection proxy. |
| POST | `/api/v1/search` | `DomainToken` + `MLSAccess` | `bridge.Handler.Search` | Hybrid structured MLS search service. |
| GET | `/api/v1/bridge/stats` | `DomainToken` + `MLSAccess` | `bridge.Handler.Stats` | Replication/mirror stats snapshot. |
| POST | `/api/v1/comps/run` | `DomainToken` + `MLSAccess` | `comps.Handler.Run` | Comps/investor/BPO/home-value engine. |
| GET | `/api/v1/admin/monitoring` | Session cookie (`session_id`) | `dashboard.Handler.MonitoringJSON` | Monitoring JSON for authenticated operators. |
| POST | `/api/v1/admin/flood-enrich` | Session cookie (`session_id`) | `admin.FloodHandler.Enrich` | Enqueue FEMA flood enrichment jobs. |

## HTML/session/static routes (non-Yaak scope)

These routes are part of the application surface but are intentionally not in the API collection because they are page/form/session flows.

| Method | Path | Auth | Handler | Purpose |
|---|---|---|---|---|
| GET | `/` | None | `marketing.Handler.Home` | Marketing/home landing page. |
| USE | `/static/*` | None | static filesystem middleware | Serve embedded static assets. |
| GET | `/login` | None | `dashboard.Handler.LoginForm` | Render login form page. |
| POST | `/login` | Form credentials | `dashboard.Handler.Login` | Create dashboard session and redirect to monitoring. |
| GET | `/logout` | Session (if present) | `dashboard.Handler.Logout` | Destroy session and redirect to login. |
| GET | `/dashboard` | Session required | `dashboard.Handler.DashboardHome` | Redirect dashboard root to monitoring page. |
| GET | `/dashboard/monitoring` | Session required | `dashboard.Handler.MonitoringPage` | Render monitoring dashboard HTML. |
| GET | `/dashboard/monitoring/data` | Session required | `dashboard.Handler.MonitoringJSON` | Monitoring JSON used by dashboard UI. |
| GET | `/dashboard/domains` | Session required | `dashboard.Handler.DomainsPage` | Render domain/token management page. |
| GET | `/dashboard/setup` | Session required | `dashboard.Handler.redirectToDomains` | Legacy setup route redirect to domains. |
| GET | `/dashboard/api-keys` | Session required | `dashboard.Handler.redirectToDomains` | Legacy API keys route redirect to domains. |
| GET | `/dashboard/invite` | Session + admin required | `dashboard.Handler.InvitePage` | Render invitation management page. |
| POST | `/dashboard/domains` | Session required | `dashboard.Handler.StoreDomain` | Provision domain bundle + datasets. |
| POST | `/dashboard/domains/:id/verify-txt` | Session required | `dashboard.Handler.VerifyTXT` | Verify DNS TXT ownership for domain. |
| POST | `/dashboard/domains/:id/regenerate-token` | Session required | `dashboard.Handler.RegenerateDomainToken` | Rotate/reissue domain token. |
| POST | `/dashboard/domains/:id/delete` | Session required | `dashboard.Handler.DeleteDomainBundle` | Delete domain bundle (production root flow). |
| POST | `/dashboard/api-tokens` | Session required | `dashboard.Handler.deprecateManualToken` | Deprecated; redirects to domain-managed token flow. |
| POST | `/dashboard/api-tokens/staging` | Session required | `dashboard.Handler.deprecateManualToken` | Deprecated; redirects to domain-managed token flow. |
| DELETE | `/dashboard/api-tokens/:id` | Session required | `dashboard.Handler.RevokeToken` | Revoke token by id. |
| POST | `/dashboard/api-tokens/:id/revoke` | Session required | `dashboard.Handler.RevokeToken` | Form-compatible token revoke endpoint. |
| POST | `/dashboard/invitations` | Session + admin required | `dashboard.Handler.CreateInvitation` | Create one-time invitation link. |
| GET | `/invite/:token` | None | `dashboard.Handler.InviteRegisterForm` | Render invite acceptance form. |
| POST | `/invite/:token` | Form payload | `dashboard.Handler.AcceptInvitation` | Accept invite and create account. |

## Coverage summary

- Registered routes in code: 61 route registrations (including static middleware `USE /static/*` and both JSON + HTML flows).
- OpenAPI collection coverage: 40 API operations (all JSON endpoints, excluding HTML/static/session pages by design).
