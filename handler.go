package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/zulkou/chirpy/internal/auth"
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
        Password string `json:"password"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := reqStruct{}
    err := decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to decode user input")
        return
    }

    hashedPassword, err := auth.HashPassword(reqData.Password)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
        return
    }

    resp, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
        Email: reqData.Email,
        HashedPassword: hashedPassword,
    })
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create user")
        return
    }

    createdUser := User{
        ID: resp.ID,
        CreatedAt: resp.CreatedAt,
        UpdatedAt: resp.UpdatedAt,
        Email: resp.Email,
        IsChirpyRed: resp.IsChirpyRed,
    }

    respondWithJSON(w, http.StatusCreated, createdUser)
    return

}

func (cfg *apiConfig) createChirpHandler(w http.ResponseWriter, r *http.Request) {
    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Failed to get token")
        return
    }

    userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Token missmatch on validation")
        return
    }

    type userChirp struct {
        Body string `json:"body"`
        UserID uuid.UUID `json:"user_id"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := userChirp{}
    err = decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to decode user input")
        return
    }

    reqData.UserID = userID

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

func (cfg *apiConfig) loginUserHandler(w http.ResponseWriter, r *http.Request) {
    type loginData struct {
        Email string `json:"email"`
        Password string `json:"password"`
        ExpiresInSeconds time.Duration `json:"expires_in_seconds"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := loginData{}
    err := decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to process input")
        return
    }

    user, err := cfg.db.GetUserByEmail(context.Background(), reqData.Email)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
        return
    }

    err = auth.CheckPasswordHash(user.HashedPassword, reqData.Password)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Incorrect email or password")
        return
    }

    var expiresIn time.Duration
    if reqData.ExpiresInSeconds != 0 {
        expiresIn = reqData.ExpiresInSeconds * time.Second
    } else {
        expiresIn = 3600 * time.Second
    }

    jwtToken, err := auth.MakeJWT(user.ID, cfg.jwtSecret, expiresIn)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create auth token")
        return
    }

    randToken, err := auth.MakeRefreshToken()
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
        return
    }

    refreshToken, err := cfg.db.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{
        Token: randToken,
        UserID: user.ID,
        ExpiresAt: time.Now().AddDate(0, 0, 60),
        RevokedAt: sql.NullTime{},
    })
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
        return
    }

    loggedUser := User{
        ID: user.ID,
        CreatedAt: user.CreatedAt,
        UpdatedAt: user.UpdatedAt,
        Email: user.Email,
        IsChirpyRed: user.IsChirpyRed,
        Token: jwtToken,
        RefreshToken: refreshToken.Token,
    }

    respondWithJSON(w, http.StatusOK, loggedUser)
    return
}

func (cfg *apiConfig) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Auth header not found")
        return
    }

    refreshToken, err := cfg.db.GetRefreshTokenByToken(context.Background(), token)
    if err != nil || refreshToken.RevokedAt.Valid || time.Now().After(refreshToken.ExpiresAt) {
        respondWithError(w, http.StatusUnauthorized, "Refresh token does not exist, is revoked, or is expired")
        return
    }

    jwtToken, err := auth.MakeJWT(refreshToken.UserID, cfg.jwtSecret, 1 * time.Hour)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to create new token")
        return
    }

    newToken := struct {
        Token string `json:"token"`
    }{
        Token: jwtToken,
    }

    respondWithJSON(w, http.StatusOK, newToken)
    return
}

func (cfg *apiConfig) revokeTokenHandler(w http.ResponseWriter, r *http.Request) {
    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Auth header not found")
        return
    }

    err = cfg.db.UpdateRevokeToken(context.Background(), token)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Token did not exists")
        return
    }

    respondWithJSON(w, http.StatusNoContent, nil)
    return
}

func (cfg *apiConfig) updateUserHandler(w http.ResponseWriter, r *http.Request) {
    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Failed to get token")
        return
    }

    userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Token missmatch on validation")
        return
    }

    type newData struct {
        Email string `json:"email"`
        Password string `json:"password"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := newData{}
    err = decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to decode input")
        return
    }

    hashedPassword, err := auth.HashPassword(reqData.Password)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
    }

    newUserData, err := cfg.db.UpdateUserByID(context.Background(), database.UpdateUserByIDParams{
        ID: userID,
        Email: reqData.Email,
        HashedPassword: hashedPassword,
    })

    newUser := User{
        ID: newUserData.ID,
        CreatedAt: newUserData.CreatedAt,
        UpdatedAt: newUserData.UpdatedAt,
        Email: newUserData.Email,
        IsChirpyRed: newUserData.IsChirpyRed,
    }

    respondWithJSON(w, http.StatusOK, newUser)
    return
}

func (cfg *apiConfig) deleteChirpByIDHandler (w http.ResponseWriter, r *http.Request) {
    stringID := r.PathValue("chirpID")
    chirpID, err := uuid.Parse(stringID)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to parse chirp id")
    }
    chirp, err := cfg.db.GetChirpByID(context.Background(), chirpID)
    if err != nil {
        respondWithError(w, http.StatusNotFound, "Failed to fetch chirp")
        return
    }

    token, err := auth.GetBearerToken(r.Header)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Failed to get token")
        return
    }

    userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
    if err != nil {
        respondWithError(w, http.StatusUnauthorized, "Token missmatch")
        return
    }

    if userID != chirp.UserID {
        respondWithError(w, http.StatusForbidden, "User unauthorized")
        return
    }

    err = cfg.db.DeleteChirpByID(context.Background(), chirp.ID)
    if err != nil {
        respondWithError(w, http.StatusNotFound, "Chirp not found")
        return
    }

    respondWithJSON(w, http.StatusNoContent, nil)
    return
}

func (cfg *apiConfig) upgradeUserHandler(w http.ResponseWriter, r *http.Request) {
    type webReq struct {
        Event string `json:"event"`
        Data struct {
            UserID string `json:"user_id"`
        } `json:"data"`
    }

    decoder := json.NewDecoder(r.Body)
    reqData := webReq{}
    err := decoder.Decode(&reqData)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to decode request")
        return
    }

    userID, err := uuid.Parse(reqData.Data.UserID)
    if err != nil {
        respondWithError(w, http.StatusInternalServerError, "Failed to parse given id")
        return
    }

    if reqData.Event != "user.upgraded" {
        respondWithJSON(w, http.StatusNoContent, nil)
        return
    }

    err = cfg.db.UpgradeUserRedChirpy(context.Background(), userID)
    if err != nil {
        respondWithError(w, http.StatusNotFound, "User not found")
        return
    }

    respondWithJSON(w, http.StatusNoContent, nil)
    return
}
