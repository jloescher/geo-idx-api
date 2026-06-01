package dashboard

import (
	"strings"

	"github.com/quantyralabs/idx-api/internal/web"
)

// NavPage identifies the active dashboard section.
type NavPage string

const (
	NavMonitoring NavPage = "monitoring"
	NavDomains    NavPage = "domains"
	NavInvite     NavPage = "invite"
	NavMCPKeys    NavPage = "mcp-keys"
	NavAdminTokens  NavPage = "admin-tokens"
	NavOAuthClients NavPage = "oauth-clients"
	NavOAuthTokens  NavPage = "oauth-tokens"
)

func renderDashboardNav(active NavPage, isAdmin bool) string {
	items := []struct {
		id    NavPage
		label string
		href  string
	}{
		{NavMonitoring, "Monitoring", "/dashboard/monitoring"},
		{NavDomains, "Domains", "/dashboard/domains"},
	}
	if isAdmin {
		items = append(items, struct {
			id    NavPage
			label string
			href  string
		}{NavInvite, "Invite user", "/dashboard/invite"})
		items = append(items, struct {
			id    NavPage
			label string
			href  string
		}{"mcp-keys", "MCP Keys", "/dashboard/mcp-keys"})
		items = append(items, struct {
			id    NavPage
			label string
			href  string
		}{NavAdminTokens, "Admin Tokens", "/dashboard/admin-tokens"})
		items = append(items, struct {
			id    NavPage
			label string
			href  string
		}{NavOAuthClients, "OAuth Clients", "/dashboard/oauth-clients"})
		items = append(items, struct {
			id    NavPage
			label string
			href  string
		}{NavOAuthTokens, "Active OAuth Tokens", "/dashboard/oauth-tokens"})
	}
	var b strings.Builder
	b.WriteString(`<nav class="dashboard-nav" aria-label="Dashboard sections"><ul class="dashboard-nav-list">`)
	for _, item := range items {
		cls := "dashboard-nav-link"
		if item.id == active {
			cls += " is-active"
		}
		aria := ""
		if item.id == active {
			aria = ` aria-current="page"`
		}
		b.WriteString(`<li><a class="`)
		b.WriteString(cls)
		b.WriteString(`" href="`)
		b.WriteString(item.href)
		b.WriteString(`"`)
		b.WriteString(aria)
		b.WriteString(`>`)
		b.WriteString(web.Esc(item.label))
		b.WriteString(`</a></li>`)
	}
	b.WriteString(`</ul></nav>`)
	return b.String()
}

func renderDashboardPage(title string, active NavPage, isAdmin bool, body string, monitoringJS bool) string {
	nav := renderDashboardNav(active, isAdmin)
	return web.DashboardPage(title, nav, body, monitoringJS)
}
