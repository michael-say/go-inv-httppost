package store

import (
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

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
	body, err := ioutil.ReadFile(path)
	if err != nil {
		log.Println("404 not foung", path)
		http.NotFound(w, r)
		return
	}
	w.Write(body)
}
