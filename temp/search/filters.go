package search

import (
	"fmt"
	"strings"

	"github.com/xotec-solutions/xotec-datalayer/src/internal/db"
)

type queryBuilder struct {
	registry *db.Registry
	clauses  []string
	args     []any
	argIndex int
}

func newQueryBuilder(registry *db.Registry) *queryBuilder {
	return &queryBuilder{
		registry: registry,
		clauses:  make([]string, 0, 16),
		args:     make([]any, 0, 16),
		argIndex: 1,
	}
}

func (b *queryBuilder) addClause(name string, args ...any) {
	fragment := strings.TrimSpace(b.registry.SQL(db.QueryName(name)))
	if fragment == "" {
		return
	}
	if len(args) > 0 {
		fragment = replaceArgs(fragment, b.argIndex, len(args))
		b.argIndex += len(args)
		b.args = append(b.args, args...)
	}
	fragment = extractClause(fragment)
	if fragment == "" {
		return
	}
	b.clauses = append(b.clauses, fragment)
}

func (b *queryBuilder) addGroupOr(clauses []string) {
	if len(clauses) == 0 {
		return
	}
	group := "(" + strings.Join(clauses, " OR ") + ")"
	b.clauses = append(b.clauses, group)
}

func (b *queryBuilder) whereClause() string {
	if len(b.clauses) == 0 {
		return ""
	}
	return "AND " + strings.Join(b.clauses, " AND ")
}

func (b *queryBuilder) argsSlice() []any {
	return b.args
}

func replaceArgs(fragment string, startIndex int, argCount int) string {
	if argCount == 0 {
		return fragment
	}
	result := fragment
	// Two-pass replacement: using $N→$M directly with strings.ReplaceAll can
	// corrupt already-placed values when startIndex >= 10 (e.g. replacing
	// $1→$11 also matches the "$1" prefix inside an already-placed "$12").
	// Pass 1: $N → intermediate placeholder (high-to-low prevents $1 matching $12).
	for i := argCount; i >= 1; i-- {
		old := fmt.Sprintf("$%d", i)
		tmp := fmt.Sprintf("@@%d@@", startIndex+i-1)
		result = strings.ReplaceAll(result, old, tmp)
	}
	// Pass 2: placeholders → final $N values.
	for i := 1; i <= argCount; i++ {
		tmp := fmt.Sprintf("@@%d@@", startIndex+i-1)
		fin := fmt.Sprintf("$%d", startIndex+i-1)
		result = strings.ReplaceAll(result, tmp, fin)
	}
	return result
}

func extractClause(fragment string) string {
	trimmed := strings.TrimSpace(strings.TrimSuffix(fragment, ";"))
	// Strip leading SQL comments and blank lines so the SELECT 1 check works
	// even when the registry includes description comments above the query.
	lines := strings.Split(trimmed, "\n")
	start := 0
	for start < len(lines) {
		line := strings.TrimSpace(lines[start])
		if line == "" || strings.HasPrefix(line, "--") {
			start++
			continue
		}
		break
	}
	if start > 0 && start < len(lines) {
		trimmed = strings.TrimSpace(strings.Join(lines[start:], "\n"))
	}
	upper := strings.ToUpper(trimmed)
	if strings.HasPrefix(upper, "SELECT 1") {
		idx := strings.Index(upper, "WHERE")
		if idx != -1 {
			return strings.TrimSpace(trimmed[idx+len("WHERE"):])
		}
	}
	return trimmed
}
