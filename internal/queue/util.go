package queue

// quoteIdent quotes a PostgreSQL identifier (channel names we control).
func quoteIdent(name string) string {
	return `"` + name + `"`
}
