package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	store "./store"
)

func getPort() int {
	portenv := os.Getenv("SRV_PORT")
	if len(portenv) > 0 {
		port, err := strconv.Atoi(portenv)
		if err != nil {
			log.Fatal("Incorrect env SRV_PORT value")
		}
		return port
	}
	return 8090
}

func main() {

	port := getPort()

	http.HandleFunc("/bin/", store.BinHandler)
	http.HandleFunc("/static/", store.StaticHandler)
	log.SetOutput(os.Stdout)
	log.Println(fmt.Sprintf("Starting server at port %d", port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil), "Server stopped")

}
