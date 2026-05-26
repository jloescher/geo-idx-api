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
		b.WriteString(`<li><a class="` + cls + `" href="` + item.href + `"` + aria + `>` + web.Esc(item.label) + `</a></li>`)
	}
	b.WriteString(`</ul></nav>`)
	return b.String()
}

func renderDashboardPage(title string, active NavPage, isAdmin bool, body string, monitoringJS bool) string {
	nav := renderDashboardNav(active, isAdmin)
	return web.DashboardPage(title, nav, body, monitoringJS)
}
