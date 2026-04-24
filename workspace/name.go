package workspace

import (
	"regexp"
	"strings"
)

var nonAlphanumRegex = regexp.MustCompile(`[^a-z0-9]+`)

// Name derives a workspace name from a session name and session UUID.
// Format: "<sanitized-name>-<short-id>", e.g. "fix-login-a1b2c3d4".
// When the name sanitizes to empty (e.g. all punctuation), only the short id is used.
func Name(sessionName, sessionID string) string {
	short := shortID(sessionID)
	sanitized := sanitize(sessionName)
	if sanitized == "" {
		return short
	}
	return sanitized + "-" + short
}

func sanitize(name string) string {
	name = strings.ToLower(name)
	name = nonAlphanumRegex.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	if len(name) > 24 {
		name = strings.TrimRight(name[:24], "-")
	}
	return name
}

func shortID(sessionID string) string {
	clean := strings.ReplaceAll(sessionID, "-", "")
	if len(clean) < 8 {
		return clean
	}
	return clean[:8]
}
