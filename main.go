package main

import (
	"net/http"
)

const LogsDir string = ".log"

func main() {
	server := http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: RequestHandler(),
	}
	server.ListenAndServe()
}
