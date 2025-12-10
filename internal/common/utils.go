package common

import "strings"

// hasAny returns true if s contains any of the substrings.
func HasAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
