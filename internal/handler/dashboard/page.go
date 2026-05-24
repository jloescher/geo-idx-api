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
	Domains             []DomainRow
	Tokens              []TokenRow
	Feeds               []dom.FeedDefinition
	IsAdmin             bool
	Setup               repository.UserSetupStats
	DomainError         string
	SubmittedHost       string
	MonitoringBootstrap *dashsvc.Snapshot
	ProvisionFlash      *dashsvc.ProvisionResult
	VerifyError         string
	VerifySuccess       string
}

// DomainRow is a domain list entry.
type DomainRow struct {
	ID        int64
	Slug      string
	Status    string
	TXTName   string
	TXTValue  string
	IsStaging bool
}

// TokenRow is an API token list entry.
type TokenRow struct {
	ID        int64
	Name      string
	Created   string
	LastUsed  string
	NeverUsed bool
}

func renderMonitoringPage(data PageData) string {
	return renderDashboardPage("Monitoring", NavMonitoring, data.IsAdmin, renderMonitoringBody(data), true)
}

func renderSetupPage(data PageData) string {
	return renderDashboardPage("Setup", NavSetup, data.IsAdmin, renderSetupBody(data), false)
}

func renderAPIKeysPage(data PageData) string {
	return renderDashboardPage("API keys", NavAPIKeys, data.IsAdmin, renderAPIKeysBody(data), false)
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
<div id="monitoring-error" class="alert-danger" hidden role="alert">
<span id="monitoring-error-text"></span>
<button type="button" id="monitoring-retry" class="btn btn-sm btn-secondary">Retry</button>
</div>
<div id="monitoring-panel" role="region" aria-live="polite" aria-busy="true">
<div class="monitoring-skeleton-grid" id="monitoring-skeleton">
<div class="metric-skeleton"></div><div class="metric-skeleton"></div><div class="metric-skeleton"></div><div class="metric-skeleton"></div>
</div>
<div id="monitoring-content" hidden></div>
</div>
<details id="monitoring-queues" class="monitoring-queues" hidden>
<summary>Queue job types (pending)</summary>
<pre id="monitoring-queue-detail" class="queue-detail"></pre>
</details>
</section>`
}

func renderSetupBody(data PageData) string {
	var b strings.Builder
	b.WriteString(`<section class="card"><h1>Setup</h1>`)
	b.WriteString(renderSetupProgress(data.Setup))
	b.WriteString(`<p>Add a domain, publish TXT records, and verify DNS before your first API call.</p>`)

	if data.VerifySuccess != "" {
		b.WriteString(`<div class="alert-success" role="status">` + web.Esc(data.VerifySuccess) + `</div>`)
	}
	if data.VerifyError != "" {
		b.WriteString(`<div class="form-error" role="alert">` + web.Esc(data.VerifyError) + `</div>`)
	}
	if data.ProvisionFlash != nil {
		b.WriteString(renderProvisionVerifyPanel(data.ProvisionFlash))
	}

	b.WriteString(`<div id="domains" class="setup-section">`)
	b.WriteString(`<h2>Your domains</h2>`)
	if len(data.Domains) == 0 {
		b.WriteString(`<div class="empty-state">
<p>No domains yet</p>
<p class="empty-hint">Use the form below to register production + staging hostnames and receive TXT verification records.</p>
</div>`)
	} else {
		b.WriteString(`<ul class="domain-list domain-verify-list">`)
		for _, d := range data.Domains {
			b.WriteString(renderDomainVerifyRow(d))
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`</div>`)

	b.WriteString(`<div id="add-domain" class="setup-section"><h2>Add domain</h2>
<p>Creates production + staging hostnames, API keys, and TXT verification records in one step.</p>`)
	if data.DomainError != "" {
		b.WriteString(`<div class="form-error" role="alert">` + web.Esc(data.DomainError) + `</div>`)
	}
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

func renderProvisionVerifyPanel(result *dashsvc.ProvisionResult) string {
	var b strings.Builder
	b.WriteString(`<section id="verify" class="card verify-panel" aria-labelledby="verify-heading">
<h2 id="verify-heading">Verify DNS</h2>
<p class="save-warning"><strong>Save your API keys now</strong> — they will not be shown again after you leave this page.</p>
<ol class="next-steps">
<li>Copy Production and Staging tokens below</li>
<li>Publish both TXT records at your DNS host</li>
<li>Click <strong>Verify TXT</strong> for each hostname</li>
</ol>
<h3>Production token</h3>
<div class="token-box" id="token-production">` + web.Esc(result.ProductionToken) + `</div>
<button type="button" class="btn btn-secondary btn-sm" data-copy="#token-production">Copy</button>
<h3>Staging token</h3>
<div class="token-box" id="token-staging">` + web.Esc(result.StagingToken) + `</div>
<button type="button" class="btn btn-secondary btn-sm" data-copy="#token-staging">Copy</button>
<h3>Production DNS (` + web.Esc(result.ProdDomain.DomainSlug) + `)</h3>
<div class="txt-block"><span class="txt-label">Host</span> <code class="txt-value">` + web.Esc(result.ProdDomain.TXTVerificationName) + `</code></div>
<div class="txt-block"><span class="txt-label">Value</span> <code class="txt-value">` + web.Esc(result.ProdDomain.TXTVerificationValue) + `</code></div>
<form method="post" action="/dashboard/domains/` + strconv.FormatInt(result.ProdDomain.ID, 10) + `/verify-txt" class="verify-form">
<button type="submit" class="btn btn-primary btn-sm">Verify production TXT</button>
</form>
<h3>Staging DNS (` + web.Esc(result.StagingDomain.DomainSlug) + `)</h3>
<div class="txt-block"><span class="txt-label">Host</span> <code class="txt-value">` + web.Esc(result.StagingDomain.TXTVerificationName) + `</code></div>
<div class="txt-block"><span class="txt-label">Value</span> <code class="txt-value">` + web.Esc(result.StagingDomain.TXTVerificationValue) + `</code></div>
<form method="post" action="/dashboard/domains/` + strconv.FormatInt(result.StagingDomain.ID, 10) + `/verify-txt" class="verify-form">
<button type="submit" class="btn btn-primary btn-sm">Verify staging TXT</button>
</form>
</section>`)
	return b.String()
}

func renderDomainVerifyRow(d DomainRow) string {
	badge := "badge-pending"
	if d.Status == "verified" || d.Status == "verified_ghl" {
		badge = "badge-verified"
	}
	liClass := ""
	if d.IsStaging {
		liClass = ` class="domain-staging"`
	}
	var b strings.Builder
	b.WriteString(`<li` + liClass + `><div class="domain-row-main"><strong>` + web.Esc(d.Slug) + `</strong> <span class="badge ` + badge + `">` + web.Esc(d.Status) + `</span></div>`)
	if d.Status != "verified" && d.Status != "verified_ghl" && d.TXTName != "" {
		b.WriteString(`<div class="txt-block"><span class="txt-label">Host</span> <code class="txt-value">` + web.Esc(d.TXTName) + `</code></div>
<div class="txt-block"><span class="txt-label">Value</span> <code class="txt-value">` + web.Esc(d.TXTValue) + `</code></div>`)
	}
	if d.Status != "verified" && d.Status != "verified_ghl" {
		b.WriteString(`<form method="post" action="/dashboard/domains/` + strconv.FormatInt(d.ID, 10) + `/verify-txt" class="verify-form">
<button type="submit" class="btn btn-sm btn-primary" aria-label="Verify TXT for ` + web.Esc(d.Slug) + `">Verify TXT</button></form>`)
	}
	b.WriteString(`</li>`)
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
<li class="%s"><a href="/dashboard/setup#add-domain">Add domain</a></li>
<li class="%s"><a href="/dashboard/setup#verify">Verify DNS</a></li>
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

func renderAPIKeysBody(data PageData) string {
	var b strings.Builder
	b.WriteString(`<section class="card"><h1>API keys</h1>`)
	if len(data.Tokens) == 0 {
		b.WriteString(`<div class="empty-state">
<p>No API keys yet</p>
<p class="empty-hint">Adding a domain on Setup creates Production and Staging keys automatically.</p>
<a class="btn btn-secondary" href="/dashboard/setup">Go to Setup</a>
</div>`)
	} else {
		b.WriteString(`<ul class="domain-list token-list">`)
		for _, t := range data.Tokens {
			used := t.LastUsed
			if t.NeverUsed {
				used = `<span class="muted-inline">Never used</span>`
			} else {
				used = web.Esc(t.LastUsed)
			}
			b.WriteString(`<li><strong>` + web.Esc(t.Name) + `</strong> <span class="badge badge-pending">` + web.Esc(t.Created) + `</span>
<span class="token-last-used">Last used: ` + used + `</span>
<form method="post" action="/dashboard/api-tokens/` + strconv.FormatInt(t.ID, 10) + `/revoke" onsubmit="return confirm('Revoke this token?');">
<button type="submit" class="btn btn-sm btn-secondary" aria-label="Revoke ` + web.Esc(t.Name) + ` token">Revoke</button></form></li>`)
		}
		b.WriteString(`</ul>`)
	}
	b.WriteString(`<p class="section-note">Domain bundles on <a href="/dashboard/setup">Setup</a> mint Production + Staging keys.</p>
<form method="post" action="/dashboard/api-tokens/staging" class="inline-form inline-form-compact">
<button type="submit" class="btn btn-sm btn-secondary">Create Staging token (advanced)</button>
</form>
<form method="post" action="/dashboard/api-tokens" class="inline-form">
<label>Custom token name <input name="name" type="text" placeholder="Custom" required></label>
<button type="submit" class="btn btn-sm btn-secondary">Create token</button>
</form></section>`)
	return b.String()
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
