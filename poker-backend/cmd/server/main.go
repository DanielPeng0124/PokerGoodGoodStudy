package main

import (
	"log"
	"net/http"
	"os"

	"poker-backend/internal/api"
	"poker-backend/internal/room"
)

func main() {
	addr := getenv("ADDR", ":8080")
	manager := room.NewManager()
	r := api.NewRouter(manager)

	log.Printf("poker backend listening on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatal(err)
	}
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
