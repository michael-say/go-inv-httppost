package store

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type staticData struct {
	MichaelQuota int64
	JohnQuota    int64
	AppQuota     int64
}

// StaticHandler handels request for static content
func StaticHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Unsupported method", http.StatusMethodNotAllowed)
	}
	log.Println("HTTP GET", r.URL.Path)

	pwd, err := getHome()
	if err != nil {
		log.Println("500 unable to read current dir: ", pwd)
		http.Error(w, "Unable to read current dir", http.StatusInternalServerError)
		return
	}

	title := strings.TrimSpace(r.URL.Path[len("/static/"):])
	if len(title) == 0 {
		title = "index.html"
	}
	path := filepath.Join(pwd, "resources", "www", title)

	var jq int64 = 0
	var mq int64 = 0
	var aq int64 = 0
	if strings.HasSuffix(path, "index.html") {
		qk := newTCPQuotaKeeper()
		jq, err = qk.getUserQuota(&dbUserID{1}, &dbAppWorkspace{"app1", "1234"})
		mq, err = qk.getUserQuota(&dbUserID{2}, &dbAppWorkspace{"app1", "1234"})
		aq, err = qk.getAppQuota(&dbAppWorkspace{"app1", "1234"})
	}

	if err != nil {
		log.Println("500 unable to get user quota: ", err.Error())
		http.Error(w, "500 unable to get user quota", http.StatusInternalServerError)
		return
	}

	data := staticData{
		JohnQuota:    jq,
		MichaelQuota: mq,
		AppQuota:     aq,
	}

	body, err := executeTemplateToFile(path, data)
	if err != nil {
		log.Println("404 not found", path, err.Error())
		http.NotFound(w, r)
		return
	}
	w.Write(body.Bytes())
}
