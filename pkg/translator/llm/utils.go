package llm

import "encoding/json"

// defaultString returns the value if non-empty, otherwise returns the default.
func defaultString(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// toFloat64 coerces a value pulled from a map[string]interface{} options bag to
// a float64. It handles the concrete numeric types that JSON decoding, config
// loading, and hand-built option maps actually produce (float64, float32, the
// integer kinds, and json.Number). Returns ok=false for nil or non-numeric
// values so callers fall back to their default instead of panicking on an
// unchecked type assertion.
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// toInt coerces a value pulled from an options bag to an int. JSON decoding
// yields float64 for numbers, so an int-only assertion silently drops a
// JSON-configured value; this accepts the numeric kinds and json.Number.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			// Fall back to float (e.g. "512.0") then truncate.
			f, ferr := n.Float64()
			if ferr != nil {
				return 0, false
			}
			return int(f), true
		}
		return int(i), true
	default:
		return 0, false
	}
}
