package store

import (
	"bytes"
	"os"
	"strings"
	"text/template"
)

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

func executeTemplateToFile(filePath string, tmplExecutor interface{}) (*bytes.Buffer, error) {
	t, err := template.ParseFiles(filePath)
	if err != nil {
		return nil, err
	}
	var tmplBuffer bytes.Buffer
	err = t.Execute(&tmplBuffer, tmplExecutor)
	if err != nil {
		return nil, err
	}
	return &tmplBuffer, nil
}

func isContentTypeAllowed(contentType string) bool {
	for _, ct := range allowedContentTypes {
		if strings.EqualFold(ct, contentType) {
			return true
		}
	}
	return false
}

func getHome() (wd string, err error) {
	wd = os.Getenv("SRV_HOME")
	if len(wd) == 0 {
		wd, err = os.Getwd()
	}
	return wd, err
}

func getResourcesPath() {

}
