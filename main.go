package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cfg.fileserverHits.Add(1)

        next.ServeHTTP(w, r)
    })
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
    hits := cfg.fileserverHits.Load()

    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
    message := ([]byte(fmt.Sprintf("Hits: %v", hits)))
    _, err := w.Write(message)
    if err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
    cfg.fileserverHits.Store(0)

    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
    message := ([]byte(fmt.Sprintf("Hit resetted")))
    _, err := w.Write(message)
    if err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(200)
    message := ([]byte("OK"))
    _, err := w.Write(message)
    if err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func main() {
    mux := http.NewServeMux()
    apiCfg := &apiConfig{}

    server := &http.Server{
        Addr: ":8080",
        Handler: mux,
    }

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./app")))))
    mux.HandleFunc("/healthz", healthzHandler)
    mux.HandleFunc("/metrics", apiCfg.metricsHandler)
    mux.HandleFunc("/reset", apiCfg.resetHandler)

    fmt.Println("Server starting...")
    err := server.ListenAndServe()
    if err != nil {
        log.Fatalf("Failed to start the server: %v", err)
    }
}
