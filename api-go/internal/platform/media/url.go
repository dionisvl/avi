// Package media provides helpers for building public media URLs.
package media

import "strings"

// URL builds a public URL for a stored object. When objectKey is already an
// absolute URL (http:// or https://), it is returned as-is — this lets seed
// fixtures point directly at external placeholder images. Otherwise the object
// key is joined onto the storage base URL.
func URL(baseURL, objectKey string) string {
	if strings.HasPrefix(objectKey, "http://") || strings.HasPrefix(objectKey, "https://") {
		return objectKey
	}
	return baseURL + "/" + objectKey
}
