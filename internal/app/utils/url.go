// Package utils contains useful functions
package utils

import "net/url"

// IsURL Is a helper function that checks if the string is a valid URL
func IsURL(payload string) bool {
	parsedURL, err := url.Parse(payload)
	if err != nil {
		return false
	}
	return parsedURL.Scheme == "https" || parsedURL.Scheme == "http"
}
