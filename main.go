package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"radio/internal/station"
	"radio/internal/web"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	mgr, err := station.NewManager(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	handler := web.NewHandler(mgr, mgr.Port(), mgr.MusicDir())

	mux := http.NewServeMux()
	mux.Handle("/", handler)

	addr := fmt.Sprintf(":%d", mgr.Port())
	log.Printf("Radio server starting on http://localhost%s", addr)
	for _, st := range mgr.AllStations() {
		log.Printf("  %s: http://localhost%s/stream/%s", st.Name(), addr, st.Mount())
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
