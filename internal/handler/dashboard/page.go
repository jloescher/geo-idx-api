package dashboard

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	dom "github.com/quantyralabs/idx-api/internal/domain"
	"github.com/quantyralabs/idx-api/internal/repository"
	dashsvc "github.com/quantyralabs/idx-api/internal/service/dashboard"
	"github.com/quantyralabs/idx-api/internal/web"
)

// PageData is server-rendered dashboard context.
type PageData struct {
	Bundles             []DomainBundle
	Feeds               []dom.FeedDefinition
	IsAdmin             bool
	Setup               repository.UserSetupStats
	DomainError         string
	SubmittedHost       string
	MonitoringBootstrap *dashsvc.Snapshot
	ProvisionFlash      *dashsvc.ProvisionResult
	TokenReveals        map[int64]string
	VerifyError         string
	VerifySuccess       string
}

// DomainBundle is a production hostname and optional staging child.
type DomainBundle struct {
	Production DomainRow
	Staging    *DomainRow
}

// DomainRow is a domain list entry with optional token metadata.
type DomainRow struct {
	ID        int64
	ParentID  int64
	Slug      string
	Status    string
	TXTName   string
	TXTValue  string
	IsStaging bool
	Token     *TokenMeta
}

// TokenMeta is dashboard-safe API key metadata.
type TokenMeta struct {
	ID        int64
	Name      string
	LastUsed  string
	NeverUsed bool
}

func renderMonitoringPage(data PageData) string {
	return renderDashboardPage("Monitoring", NavMonitoring, data.IsAdmin, renderMonitoringBody(data), true)
}

func renderDomainsPage(data PageData) string {
	return renderDashboardPage("Domains", NavDomains, data.IsAdmin, renderDomainsBody(data), false)
}

func renderInvitePage(data PageData) string {
	return renderDashboardPage("Invite user", NavInvite, data.IsAdmin, renderInviteBody(), false)
}

func renderMonitoringBody(data PageData) string {
	bootstrapJSON := "null"
	if data.MonitoringBootstrap != nil {
		if raw, err := json.Marshal(data.MonitoringBootstrap); err == nil {
			bootstrapJSON = string(raw)
		}
	}
	return `<section class="card" id="monitoring" aria-labelledby="monitoring-heading">
<script type="application/json" id="monitoring-bootstrap">` + bootstrapJSON + `</script>
<h1 id="monitoring-heading">Monitoring</h1>
<div class="monitoring-header">
<p class="monitoring-meta">Live ops metrics · PostgreSQL queue workers</p>
<div class="monitoring-controls">
<span id="monitoring-updated" class="monitoring-updated" role="status" aria-live="polite">Loading…</span>
<span class="auto-refresh-dot" aria-hidden="true" title="Auto-refresh every 30s"></span>
<button type="button" id="monitoring-refresh" class="btn btn-secondary btn-sm" aria-busy="false">Refresh</button>
</div>
</div>
<div id="monitoring-critical-strip" class="monitoring-critical-strip" hidden role="region" aria-live="polite" aria-label="Active incidents"></div>
<div class="monitoring-tabs-shell">
<div class="monitoring-tabs" role="tablist" aria-label="Monitoring categories">
<button type="button" class="monitoring-tab" role="tab" id="tab-overview" data-tab="overview" aria-controls="panel-overview" aria-selected="true" tabindex="0">Overview</button>
<button type="button" class="monitoring-tab" role="tab" id="tab-ingest" data-tab="ingest" aria-controls="panel-ingest" aria-selected="false" tabindex="-1">Ingest &amp; Sync</button>
<button type="button" class="monitoring-tab" role="tab" id="tab-queues" data-tab="queues" aria-controls="panel-queues" aria-selected="false" tabindex="-1">Queue &amp; Jobs</button>
<button type="button" class="monitoring-tab" role="tab" id="tab-data" data-tab="data" aria-controls="panel-data" aria-selected="false" tabindex="-1">Data Quality</button>
<button type="button" class="monitoring-tab" role="tab" id="tab-infra" data-tab="infra" aria-controls="panel-infra" aria-selected="false" tabindex="-1">Infrastructure</button>
<button type="button" class="monitoring-tab" role="tab" id="tab-integrations" data-tab="integrations" aria-controls="panel-integrations" aria-selected="false" tabindex="-1">Integrations</button>
<button type="button" class="monitoring-tab" role="tab" id="tab-incidents" data-tab="incidents" aria-controls="panel-incidents" aria-selected="false" tabindex="-1">Incidents</button>
</div>
</div>
<div id="monitoring-error" class="alert-danger" hidden role="alert">
<span id="monitoring-error-text"></span>
<button type="button" id="monitoring-retry" class="btn btn-sm btn-secondary">Retry</button>
</div>
<div id="monitoring-panel" role="region" aria-live="polite" aria-busy="true">
<div class="monitoring-skeleton-grid" id="monitoring-skeleton">
<div class="metric-skeleton"></div><div class="metric-skeleton"></div><div class="metric-skeleton"></div><div class="metric-skeleton"></div>
</div>
<div id="monitoring-content" hidden>
<section class="monitoring-panel active" id="panel-overview" role="tabpanel" aria-labelledby="tab-overview" data-panel="overview"></section>
<section class="monitoring-panel" id="panel-ingest" role="tabpanel" aria-labelledby="tab-ingest" data-panel="ingest" hidden></section>
<section class="monitoring-panel" id="panel-queues" role="tabpanel" aria-labelledby="tab-queues" data-panel="queues" hidden></section>
<section class="monitoring-panel" id="panel-data" role="tabpanel" aria-labelledby="tab-data" data-panel="data" hidden></section>
<section class="monitoring-panel" id="panel-infra" role="tabpanel" aria-labelledby="tab-infra" data-panel="infra" hidden></section>
<section class="monitoring-panel" id="panel-integrations" role="tabpanel" aria-labelledby="tab-integrations" data-panel="integrations" hidden></section>
<section class="monitoring-panel" id="panel-incidents" role="tabpanel" aria-labelledby="tab-incidents" data-panel="incidents" hidden></section>
</div>
</div>
<details id="monitoring-queues" class="monitoring-queues" hidden>
<summary>Queue job types (pending)</summary>
<pre id="monitoring-queue-detail" class="queue-detail"></pre>
</details>
</section>`
}

func renderDomainsBody(data PageData) string {
	var b strings.Builder
	b.WriteString(`<section class="card"><h1>Domains</h1>`)
	b.WriteString(renderSetupProgress(data.Setup))
	b.WriteString(`<p>Add a domain, publish TXT records, and manage API keys per hostname.</p>`)

	if data.VerifySuccess != "" {
		b.WriteString(`<div class="alert-success" role="status">` + web.Esc(data.VerifySuccess) + `</div>`)
	}
	if data.VerifyError != "" {
		b.WriteString(`<div class="form-error" role="alert">` + web.Esc(data.VerifyError) + `</div>`)
	}
	if data.DomainError != "" {
		b.WriteString(`<div class="form-error" role="alert">` + web.Esc(data.DomainError) + `</div>`)
	}
	if data.ProvisionFlash != nil {
		b.WriteString(renderProvisionVerifyPanel(data.ProvisionFlash, data.TokenReveals))
	}

	b.WriteString(`<div id="domains" class="setup-section">`)
	b.WriteString(`<h2>Your domains</h2>`)
	if len(data.Bundles) == 0 {
		b.WriteString(`<div class="empty-state">
<p>No domains yet</p>
<p class="empty-hint">Use the form below to register production + staging hostnames and API keys.</p>
</div>`)
	} else {
		b.WriteString(`<ul class="domain-list domain-bundle-list">`)
		for _, bundle := range data.Bundles {
			b.WriteString(renderDomainBundle(bundle, data.TokenReveals))
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div id="add-domain" class="setup-section"><h2>Add domain</h2>
<p>Creates production + staging hostnames, one API key each, and TXT verification records.</p>`)
	hostVal := web.Esc(data.SubmittedHost)
	previewHost := hostVal
	if previewHost == "" {
		previewHost = "your-domain.com"
	}
	b.WriteString(`<form method="post" action="/dashboard/domains" class="add-domain-form" id="add-domain-form">
<label>Hostname <input name="domain_slug" id="domain-hostname" type="text" placeholder="clearwaterflhouses.com" required value="` + hostVal + `"></label>
<p class="staging-preview" id="staging-preview">Staging hostname will be <code>staging.` + previewHost + `</code></p>
<fieldset class="dataset-checkboxes">
<legend>MLS datasets</legend>`)
	for i, f := range data.Feeds {
		checked := ""
		if i == 0 {
			checked = ` checked`
		}
		label := feedDisplayName(f)
		provider := feedProviderLabel(f.Provider)
		b.WriteString(fmt.Sprintf(`<label class="checkbox-row"><input type="checkbox" name="mls_datasets[]" value="%s"%s> <span class="feed-label">%s</span> <span class="feed-provider">%s</span></label>`,
			web.Esc(f.Code), checked, web.Esc(label), web.Esc(provider)))
	}
	b.WriteString(`</fieldset>
<button type="submit" class="btn btn-primary" id="add-domain-submit">Add domain &amp; verify DNS</button>
</form></div></section>`)
	return b.String()
}

func renderDomainBundle(bundle DomainBundle, reveals map[int64]string) string {
	var b strings.Builder
	b.WriteString(`<li class="domain-bundle" id="domain-bundle-` + strconv.FormatInt(bundle.Production.ID, 10) + `">`)
	b.WriteString(renderDomainHostSection(bundle.Production, reveals, true, bundle.Production.ID))
	if bundle.Staging != nil {
		b.WriteString(`<div class="domain-bundle-staging">`)
		b.WriteString(renderDomainHostSection(*bundle.Staging, reveals, false, bundle.Production.ID))
		b.WriteString(`</div>`)
	}
	b.WriteString(`<form method="post" action="/dashboard/domains/` + strconv.FormatInt(bundle.Production.ID, 10) + `/delete" class="delete-bundle-form" onsubmit="return confirm('Delete this domain and its staging hostname? API keys will stop working immediately.');">
<button type="submit" class="btn btn-sm btn-secondary">Delete domain bundle</button>
</form>`)
	b.WriteString(`</li>`)
	return b.String()
}

func renderDomainHostSection(d DomainRow, reveals map[int64]string, isProduction bool, bundleProdID int64) string {
	anchor := `domain-` + strconv.FormatInt(d.ID, 10)
	badge := "badge-pending"
	if d.Status == "verified" || d.Status == "verified_ghl" {
		badge = "badge-verified"
	}
	roleLabel := "Production"
	if d.IsStaging {
		roleLabel = "Staging"
	}
	var b strings.Builder
	b.WriteString(`<article class="domain-host-section` + stagingSectionClass(d.IsStaging) + `" id="` + anchor + `">`)
	b.WriteString(`<div class="domain-row-main"><strong>` + web.Esc(d.Slug) + `</strong> <span class="badge ` + badge + `">` + web.Esc(d.Status) + `</span> <span class="badge badge-pending">` + web.Esc(roleLabel) + `</span></div>`)

	plain := ""
	if reveals != nil {
		plain = reveals[d.ID]
	}
	b.WriteString(`<div class="domain-token-block"><h4 class="token-block-heading">API key</h4>`)
	b.WriteString(renderTokenField(d.ID, plain, d.Token))
	b.WriteString(`</div>`)

	if d.Status != "verified" && d.Status != "verified_ghl" && d.TXTName != "" {
		b.WriteString(`<div class="txt-block"><span class="txt-label">Host</span> <code class="txt-value">` + web.Esc(d.TXTName) + `</code></div>
<div class="txt-block"><span class="txt-label">Value</span> <code class="txt-value">` + web.Esc(d.TXTValue) + `</code></div>`)
	}
	if d.Status != "verified" && d.Status != "verified_ghl" {
		b.WriteString(`<form method="post" action="/dashboard/domains/` + strconv.FormatInt(d.ID, 10) + `/verify-txt" class="verify-form">
<button type="submit" class="btn btn-sm btn-primary" aria-label="Verify TXT for ` + web.Esc(d.Slug) + `">Verify TXT</button></form>`)
	}
	_ = isProduction
	_ = bundleProdID
	b.WriteString(`</article>`)
	return b.String()
}

func stagingSectionClass(isStaging bool) string {
	if isStaging {
		return " domain-staging"
	}
	return ""
}

func renderTokenField(domainID int64, plain string, meta *TokenMeta) string {
	boxID := "token-" + strconv.FormatInt(domainID, 10)
	var b strings.Builder
	if plain != "" {
		b.WriteString(`<div class="token-field">`)
		b.WriteString(`<div class="token-box token-masked" id="` + boxID + `" data-token-secret="` + web.Esc(plain) + `" data-masked="` + tokenMask + `">` + tokenMask + `</div>`)
		b.WriteString(`<div class="token-actions">`)
		b.WriteString(`<button type="button" class="btn btn-secondary btn-sm" data-token-toggle="#` + boxID + `" aria-pressed="false">Show</button>`)
		b.WriteString(`<button type="button" class="btn btn-secondary btn-sm" data-copy="#` + boxID + `">Copy</button>`)
		b.WriteString(`</div><p class="token-hint save-warning">Save this key now — it will not be shown again after you leave this page.</p></div>`)
	} else if meta != nil {
		b.WriteString(`<div class="token-field">`)
		b.WriteString(`<div class="token-box token-masked" id="` + boxID + `">` + tokenMask + `</div>`)
		b.WriteString(`<div class="token-actions">`)
		b.WriteString(`<button type="button" class="btn btn-secondary btn-sm" disabled title="Regenerate to view a new key">Show</button>`)
		b.WriteString(`<button type="button" class="btn btn-secondary btn-sm" disabled>Copy</button>`)
		b.WriteString(`</div><p class="token-hint">Regenerate to issue a new key you can copy. Last used: ` + tokenLastUsed(meta) + `</p></div>`)
	} else {
		b.WriteString(`<p class="token-hint">No API key for this domain yet.</p>`)
	}
	b.WriteString(`<form method="post" action="/dashboard/domains/` + strconv.FormatInt(domainID, 10) + `/regenerate-token" class="inline-form inline-form-compact regenerate-form" onsubmit="return confirm('Regenerate API key? The current key stops working immediately.');">
<button type="submit" class="btn btn-sm btn-secondary">Regenerate API key</button></form>`)
	return b.String()
}

const tokenMask = "idx_•••••••••••••••••••••••••••••••••"

func tokenLastUsed(meta *TokenMeta) string {
	if meta.NeverUsed {
		return "never"
	}
	return web.Esc(meta.LastUsed)
}

func renderProvisionVerifyPanel(result *dashsvc.ProvisionResult, reveals map[int64]string) string {
	revealProd := result.ProductionToken
	revealStaging := result.StagingToken
	if reveals != nil {
		if p, ok := reveals[result.ProdDomain.ID]; ok && p != "" {
			revealProd = p
		}
		if s, ok := reveals[result.StagingDomain.ID]; ok && s != "" {
			revealStaging = s
		}
	}
	var b strings.Builder
	b.WriteString(`<section id="verify" class="card verify-panel" aria-labelledby="verify-heading">
<h2 id="verify-heading">Verify DNS</h2>
<p class="save-warning"><strong>Save your API keys now</strong> — they will not be shown again after you leave this page.</p>
<ol class="next-steps">
<li>Copy each API key below (Show, then Copy)</li>
<li>Publish both TXT records at your DNS host</li>
<li>Click <strong>Verify TXT</strong> for each hostname</li>
</ol>`)
	b.WriteString(`<h3>Production (` + web.Esc(result.ProdDomain.DomainSlug) + `)</h3>`)
	b.WriteString(renderTokenField(result.ProdDomain.ID, revealProd, nil))
	b.WriteString(`<h3>Production DNS</h3>
<div class="txt-block"><span class="txt-label">Host</span> <code class="txt-value">` + web.Esc(result.ProdDomain.TXTVerificationName) + `</code></div>
<div class="txt-block"><span class="txt-label">Value</span> <code class="txt-value">` + web.Esc(result.ProdDomain.TXTVerificationValue) + `</code></div>
<form method="post" action="/dashboard/domains/` + strconv.FormatInt(result.ProdDomain.ID, 10) + `/verify-txt" class="verify-form">
<button type="submit" class="btn btn-primary btn-sm">Verify production TXT</button>
</form>`)
	b.WriteString(`<h3>Staging (` + web.Esc(result.StagingDomain.DomainSlug) + `)</h3>`)
	b.WriteString(renderTokenField(result.StagingDomain.ID, revealStaging, nil))
	b.WriteString(`<h3>Staging DNS</h3>
<div class="txt-block"><span class="txt-label">Host</span> <code class="txt-value">` + web.Esc(result.StagingDomain.TXTVerificationName) + `</code></div>
<div class="txt-block"><span class="txt-label">Value</span> <code class="txt-value">` + web.Esc(result.StagingDomain.TXTVerificationValue) + `</code></div>
<form method="post" action="/dashboard/domains/` + strconv.FormatInt(result.StagingDomain.ID, 10) + `/verify-txt" class="verify-form">
<button type="submit" class="btn btn-primary btn-sm">Verify staging TXT</button>
</form>
</section>`)
	return b.String()
}

func renderSetupProgress(s repository.UserSetupStats) string {
	step := 1
	if s.DomainCount > 0 {
		step = 2
	}
	if s.VerifiedDomainCount > 0 {
		step = 3
	}
	if s.HasAuditTraffic30d {
		step = 4
	}
	return fmt.Sprintf(`<ol class="setup-progress" aria-label="Setup progress">
<li class="%s"><a href="/dashboard/domains#add-domain">Add domain</a></li>
<li class="%s"><a href="/dashboard/domains#verify">Verify DNS</a></li>
<li class="%s"><a href="/dashboard/monitoring">First API call</a></li>
</ol>`, stepClass(1, step), stepClass(2, step), stepClass(3, step))
}

func stepClass(n, current int) string {
	if n < current {
		return "step-done"
	}
	if n == current {
		return "step-current"
	}
	return "step-upcoming"
}

func renderInviteBody() string {
	return `<section class="card"><h1>Invite user</h1>
<p>Send an invitation link to onboard a new dashboard user.</p>
<form method="post" action="/dashboard/invitations" class="inline-form">
<label>Email <input name="email" type="email" required autocomplete="email"></label>
<button type="submit" class="btn btn-primary">Send invitation</button>
</form></section>`
}

func feedDisplayName(f dom.FeedDefinition) string {
	switch strings.ToLower(f.Dataset) {
	case "stellar":
		return "Stellar"
	case "beaches":
		return "Beaches"
	default:
		if len(f.Dataset) == 0 {
			return f.Code
		}
		return strings.ToUpper(f.Dataset[:1]) + f.Dataset[1:]
	}
}

func feedProviderLabel(provider string) string {
	switch strings.ToLower(provider) {
	case "bridge":
		return "Bridge"
	case "spark":
		return "Spark"
	default:
		if len(provider) == 0 {
			return provider
		}
		return strings.ToUpper(provider[:1]) + provider[1:]
	}
}
