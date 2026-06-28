// Package logsec redacts secret-shaped substrings from log lines before they
// are persisted, so tokens or passwords that leak into build output are never
// stored in plain text.
package logsec

import "regexp"

var (
	// GitHub personal/OAuth/refresh/server tokens: ghp_, gho_, ghu_, ghs_, ghr_.
	githubToken = regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{16,}`)

	// key=value / key: value forms for common secret keys.
	keyValue = regexp.MustCompile(`(?i)\b(token|secret|password|passwd|api[_-]?key|access[_-]?key|authorization)\b(\s*[:=]\s*)("?)([^\s"',]+)`)

	// Authorization: Bearer <token>.
	bearer = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._\-]+`)
)

const redacted = "***REDACTED***"

// Mask returns s with any detected secrets replaced by a redaction marker.
func Mask(s string) string {
	s = githubToken.ReplaceAllString(s, redacted)
	s = bearer.ReplaceAllString(s, "Bearer "+redacted)
	s = keyValue.ReplaceAllString(s, `${1}${2}${3}`+redacted)
	return s
}
