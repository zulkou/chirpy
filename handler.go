package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/zulkou/chirpy/internal/database"
)

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
        respondWithError(w, http.StatusInternalServerError, "Failed to write into http response")
        return
    }
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
    if cfg.platform != "dev" {
        respondWithError(w, http.StatusForbidden, "Failed on dev auth")
        return
    }

    cfg.fileserverHits.Store(0)
    err := cfg.db.DeleteUsers(context.Background())
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to delete users")
        return
    }

    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    message := ([]byte(fmt.Sprintf("ALL RESETTED")))
    _, err = w.Write(message)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to write into http response")
        return
    }
}

func (cfg *apiConfig) healthzHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Add("Content-Type", "text/plain; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    message := ([]byte("OK"))
    _, err := w.Write(message)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to write into http response")
        return
    }
}

func (cfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {
    type reqStruct struct {
        Email string `json:"email"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := reqStruct{}
    err := decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to decode user input")
        return
    }

    resp, err := cfg.db.CreateUser(context.Background(), reqData.Email)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create user")
        return
    }

    createdUser := User{
        ID: resp.ID,
        CreatedAt: resp.CreatedAt,
        UpdatedAt: resp.UpdatedAt,
        Email: resp.Email,
    }

    respondWithJSON(w, http.StatusCreated, createdUser)
    return

}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
    type userChirp struct {
        Body string `json:"body"`
        UserID uuid.UUID `json:"user_id"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := userChirp{}
    err := decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to decode user input")
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

    resp, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
        Body: res,
        UserID: reqData.UserID,
    })
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
        return
    }

    createdChirp := Chirp{
        ID: resp.ID,
        CreatedAt: resp.CreatedAt,
        UpdatedAt: resp.UpdatedAt,
        Body: resp.Body,
        UserID: resp.UserID,
    }

    respondWithJSON(w, http.StatusCreated, createdChirp)
    return
}

func (cfg *apiConfig) getChirpsHandler(w http.ResponseWriter, r *http.Request) {
    chirps, err := cfg.db.GetChirps(context.Background())
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to fetch chirps")
        return
    }

    var chirpsSlice []Chirp
    for _, chirp := range chirps {
        chirpsSlice = append(chirpsSlice, Chirp{
            ID: chirp.ID,
            CreatedAt: chirp.CreatedAt,
            UpdatedAt: chirp.UpdatedAt,
            Body: chirp.Body,
            UserID: chirp.UserID,
        })
    }

    respondWithJSON(w, http.StatusOK, chirpsSlice)
    return
}

func (cfg *apiConfig) getChirpByIDHandler(w http.ResponseWriter, r *http.Request) {
    stringID := r.PathValue("chirpID")
    chirpID, err := uuid.Parse(stringID)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to parse string into id")
        return
    }

    chirpData, err := cfg.db.GetChirpByID(context.Background(), chirpID)
    if err != nil {
        respondWithError(w, http.StatusNotFound, fmt.Sprintf("Failed to fetch chirp with ID: %v", chirpID))
        return
    }

    chirp := Chirp{
        ID: chirpData.ID,
        CreatedAt: chirpData.CreatedAt,
        UpdatedAt: chirpData.UpdatedAt,
        Body: chirpData.Body,
        UserID: chirpData.UserID,
    }

    respondWithJSON(w, http.StatusOK, chirp)
    return
}
