package store

import "strings"

// Address represents single page metadata
type Address struct {
	App         string
	WorkspaceID int64
}

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
	chunkSize         = 512
	maxUploadFileSize = 100 << 20

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
