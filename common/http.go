package common

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// IsHTMLResponse checks if the given string appears to be an HTML response
func IsHTMLResponse(body string) bool {
	trimmed := strings.TrimSpace(body)
	return strings.HasPrefix(trimmed, "<!DOCTYPE") ||
		strings.HasPrefix(trimmed, "<html") ||
		strings.HasPrefix(trimmed, "<?xml")
}

func UnmarshalBody(r io.Reader, v any) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	// Check if response is HTML instead of JSON
	bodyStr := string(body)
	if IsHTMLResponse(bodyStr) {
		// Don't truncate HTML error responses for debugging purposes
		return fmt.Errorf("received HTML response instead of JSON: %s", bodyStr)
	}

	if err := json.Unmarshal(body, v); err != nil {
		// Provide more detailed error information for JSON parsing errors
		if strings.Contains(err.Error(), "invalid character") {
			// Try to identify the problematic part of the JSON
			return fmt.Errorf("JSON parsing error: %v. Response body starts with: %s",
				err, truncateString(bodyStr, 100))
		}
		return err
	}
	return nil
}

// truncateString shortens a string to the specified length and adds "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
