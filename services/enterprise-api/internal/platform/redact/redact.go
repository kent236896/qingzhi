// Package redact removes configured secrets from diagnostic text.
package redact

import "strings"

// Apply replaces each non-empty secret with [REDACTED].
func Apply(message string, secrets ...string) string {
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		message = strings.ReplaceAll(message, secret, "[REDACTED]")
	}
	return message
}
