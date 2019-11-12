package store

import "strings"

// PostResponseItem defines single saved binary result
type PostResponseItem struct {
	GUID     string `json:"guid"`
	FileName string `json:"fileName"`
}

// PostResponse defines POST response
type PostResponse struct {
	DiskQuota int64              `json:"diskQuota"`
	Result    []PostResponseItem `json:"result"`
}

const (
	chunkSize           = 512
	postFileFieldName   = "file"
	userIDFileFieldName = "userId"
)

var allowedContentTypes = [...]string{"application/octet-stream", "image/jpeg", "application/zip", "application/pdf", "video/avi", "audio/mpeg", "application/x-gzip", "text/plain"}

func isContentTypeAllowed(contentType string) bool {
	for _, ct := range allowedContentTypes {
		if strings.EqualFold(ct, contentType) {
			return true
		}
	}
	return false
}
