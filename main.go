package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"unicode"
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

    w.Header().Add("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    message := (fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, hits))
    _, err := w.Write([]byte(message))
    if err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
    cfg.fileserverHits.Store(0)

    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    message := ([]byte(fmt.Sprintf("Hit resetted")))
    _, err := w.Write(message)
    if err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func (cfg *apiConfig) healthzHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    message := ([]byte("OK"))
    _, err := w.Write(message)
    if err != nil {
        log.Printf("Error writing response: %v", err)
    }
}

func (cfg *apiConfig) validateChirpHandler(w http.ResponseWriter, r *http.Request) {
    type chirpData struct {
        Body string `json:"body"`
    }

    type cleanData struct {
        CleanedBody string `json:"cleaned_body"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := chirpData{}
    err := decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Something went wrong")
        return
    }

    if len(reqData.Body) > 140 {
        respondWithError(w, http.StatusBadRequest, "Chirp is too long")
        return
    }
    
    profanities := map[string]bool{
        "kerfuffle": true, 
        "sharbert": true, 
        "fornax": true,
    }

    cleanWord := func(word string) string {
        return strings.TrimFunc(word, func(r rune) bool {
            return !unicode.IsLetter(r)
        })
    }

    words := strings.Split(reqData.Body, " ")
    for idx, word := range words {
        lower := cleanWord(strings.ToLower(word))
        if profanities[lower] {
            words[idx] = "****"
        }
    }
    res := strings.Join(words, " ")

    respData := cleanData{
        CleanedBody: res,
    }

    respondWithJSON(w, http.StatusOK, respData)
    return
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

func main() {
    mux := http.NewServeMux()
    apiCfg := &apiConfig{}

    server := &http.Server{
        Addr: ":8080",
        Handler: mux,
    }

    mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir("./app")))))

    mux.HandleFunc("GET /api/healthz", apiCfg.healthzHandler)
    mux.HandleFunc("POST /api/validate_chirp", apiCfg.validateChirpHandler)

    mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
    mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)

    fmt.Println("Server starting...")
    err := server.ListenAndServe()
    if err != nil {
        log.Fatalf("Failed to start the server: %v", err)
    }
}
