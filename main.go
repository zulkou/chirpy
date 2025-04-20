package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/zulkou/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
    db *database.Queries
    platform string
    jwtSecret string
}

func main() {
    godotenv.Load()

    roles := os.Getenv("PLATFORM")
    jwtString := os.Getenv("JWT_SECRET")

    dbURL := os.Getenv("DB_URL")
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatalf("Failed to start the database: %v", err)
        return
    }
    defer db.Close()
    dbQueries := database.New(db)

    mux := http.NewServeMux()
    apiCfg := &apiConfig{
        db: dbQueries,
        platform: roles,
        jwtSecret: jwtString,
    }

    server := &http.Server{
        Addr: ":8080",
        Handler: mux,
    }

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./app")))))

    mux.HandleFunc("GET /api/healthz", apiCfg.healthzHandler)

    mux.HandleFunc("POST /api/login", apiCfg.loginUserHandler)
    mux.HandleFunc("POST /api/users", apiCfg.createUserHandler)

    mux.HandleFunc("POST /api/chirps", apiCfg.createChirpHandler)
    mux.HandleFunc("GET /api/chirps", apiCfg.getChirpsHandler)
    mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpByIDHandler)

    mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
    mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

    fmt.Println("Server starting...")
    err = server.ListenAndServe()
    if err != nil {
        log.Fatalf("Failed to start the server: %v", err)
        return
    }
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    resp, err := json.Marshal(payload)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Something went wrong")
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(resp)

}

func respondWithError(w http.ResponseWriter, code int, msg string) {
    type errorResp struct {
        Error string `json:"error"`
    }

    resp := errorResp{
        Error: msg,
    }
    data, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error marshalling error response: %s", err)
        w.Header().Set("Content-Type", "text/plain")
        w.WriteHeader(http.StatusInternalServerError)
        w.Write([]byte("Internal server error"))
        return
    }
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(data)
}
