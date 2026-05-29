package web

import (
	"fmt"
	"html"
	"strings"
)

// Page wraps body HTML with site chrome and asset links.
func Page(title, body string) string {
	return dashboardShell(title, "", body, false)
}

// DashboardPage renders dashboard chrome with section navigation.
func DashboardPage(title, navHTML, body string, _ bool) string {
	return dashboardShell(title, navHTML, body, true)
}

func dashboardShell(title, navHTML, body string, dashboardLayout bool) string {
	title = html.EscapeString(title)
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s · Quantyra IDX</title>\n", title)
	b.WriteString("<link rel=\"stylesheet\" href=\"/static/css/app.css\">\n")
	b.WriteString("<script src=\"/static/js/app.js\" defer></script>\n")
	if dashboardLayout {
		b.WriteString("<script src=\"/static/js/dashboard.js?v=20260529b\" defer></script>\n")
	}
	b.WriteString("</head>\n<body>\n")
	b.WriteString("<header class=\"site-header\">\n")
	b.WriteString("<a class=\"brand\" href=\"/\">Quantyra IDX</a>\n")
	b.WriteString("<nav class=\"site-header-actions\"><a href=\"/dashboard/monitoring\">Dashboard</a><a href=\"/logout\">Sign out</a></nav>\n")
	b.WriteString("</header>\n")
	if dashboardLayout {
		b.WriteString("<div class=\"dashboard-layout\">\n")
		if navHTML != "" {
			b.WriteString(navHTML)
		}
		b.WriteString("<main class=\"dashboard-main\">\n")
	} else {
		b.WriteString("<main>\n")
	}
	b.WriteString(body)
	if dashboardLayout {
		b.WriteString("\n</main></div>\n")
	} else {
		b.WriteString("\n</main>\n")
	}
	b.WriteString("</body>\n</html>")
	return b.String()
}

// LoginPage is the centered login layout.
func LoginPage(body string) string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Login · Quantyra IDX</title>
<link rel="stylesheet" href="/static/css/app.css">
</head>
<body>
<div class="center-page">
<div class="card">
<h1>Sign in</h1>
<p>Access your MLS domains and API keys.</p>
` + body + `
<p style="margin-top:1.5rem;font-size:0.85rem"><a href="/">Back to home</a></p>
</div></div></body></html>`
}

// Esc escapes HTML in dynamic content.
func Esc(s string) string { return html.EscapeString(s) }
