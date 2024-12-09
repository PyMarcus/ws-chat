package main

import (
	"log"
	"net/http"

	"github.com/PyMarcus/ws-chat/internal/handlers"
)

func main() {
	mux := routes()
	log.Println("Starting web server on port 8080")
	go handlers.ListenToWsChan()

	_ = http.ListenAndServe(":8080", mux)
}
