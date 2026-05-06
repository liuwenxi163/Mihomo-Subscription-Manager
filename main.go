package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dbPath := valueOr(os.Getenv("DB_PATH"), "mihomo-sub-manager.db")
	addr := valueOr(os.Getenv("ADDR"), ":49321")
	store, err := OpenStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	config := DefaultAppConfig()
	app := NewApp(store, config)
	log.Printf("listening on %s, admin user=%s", addr, config.AdminUser)
	log.Fatal(http.ListenAndServe(addr, app.routes()))
}
