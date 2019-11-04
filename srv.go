package main

import (
	"log"
	"net/http"
	"os"

	store "./store"
)

func main() {

	http.HandleFunc("/bin/", store.BinHandler)
	http.HandleFunc("/static/", store.StaticHandler)
	log.SetOutput(os.Stdout)
	log.Fatal(http.ListenAndServe(":8090", nil))

}
