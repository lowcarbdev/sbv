package internal

import "strings"

// normalizePhoneNumber removes all non-numeric characters except leading +
// and standardizes US phone numbers to include the +1 country code
// This prevents duplicate conversations due to different phone number formatting
func normalizePhoneNumber(phoneNumber string) string {
	if phoneNumber == "" {
		return ""
	}

	// Check if it starts with +
	hasPlus := strings.HasPrefix(phoneNumber, "+")

	// Remove all non-numeric characters
	var result strings.Builder
	for _, ch := range phoneNumber {
		if ch >= '0' && ch <= '9' {
			result.WriteRune(ch)
		}
	}

	normalized := result.String()
	if normalized == "" {
		return ""
	}

	// Standardize US phone numbers
	if !hasPlus {
		// 10 digits without country code - add +1 (US number)
		if len(normalized) == 10 {
			return "+1" + normalized
		}
		// 11 digits starting with 1 - add + (US number with 1 prefix)
		if len(normalized) == 11 && normalized[0] == '1' {
			return "+" + normalized
		}
		// Other lengths without + - keep as is (might be partial/invalid)
		return normalized
	}

	// Already has +, keep it
	return "+" + normalized
}
