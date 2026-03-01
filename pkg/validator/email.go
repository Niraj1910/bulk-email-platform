package validator

import (
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,4}$`)
var cleanEmailRegex = regexp.MustCompile(`[^a-zA-Z0-9.@_%+-]`)

func IsEmailValid(email string) bool {
	if email == "" {
		return false
	}
	if len(email) < 3 || len(email) > 254 {
		return false
	}

	return emailRegex.MatchString(email)
}

func NormalizeEmail(email string) string {
	email = strings.TrimSpace(email)
	// Remove all invalid characters
	// ReplaceAllString finds ANY match and replaces with empty string
	email = cleanEmailRegex.ReplaceAllString(email, "")

	// Convert to lowercase
	return strings.ToLower(email)
}

func ValidateEmail(email string) (string, bool) {
	normalized := NormalizeEmail(email)
	return normalized, IsEmailValid(normalized)
}
