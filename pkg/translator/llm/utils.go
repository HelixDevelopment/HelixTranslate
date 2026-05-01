package llm

// defaultString returns the value if non-empty, otherwise returns the default.
func defaultString(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}
