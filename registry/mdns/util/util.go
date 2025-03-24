// Package util provides utilities.
package util

import "strings"

// TrimDot is used to trim the dots from the start or end of a string.
func TrimDot(s string) string {
	return strings.Trim(s, ".")
}
