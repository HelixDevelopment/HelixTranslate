package llm

import (
	"os"
)

// getTestAPIKey retrieves API key from environment variable for testing
func getTestAPIKey(envVar string) string {
	return os.Getenv(envVar)
}