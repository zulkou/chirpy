package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return "", err
    }

    return string(hash), nil
}

func CheckPasswordHash(hash, password string) error {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    if err != nil {
        return err
    }

    return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
    claims := &jwt.RegisteredClaims{
        Issuer: "chirpy",
        IssuedAt: jwt.NewNumericDate(time.Now()),
        ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
        Subject: userID.String(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    ss, err := token.SignedString([]byte(tokenSecret))
    if err != nil {
        return "", err
    }

    return ss, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
    token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(tokenSecret), nil
    })

    if err != nil || !token.Valid{
        return uuid.Nil, err
    }

    subject, _ := token.Claims.GetSubject()
    userID, err := uuid.Parse(subject)
    if err != nil {
        return uuid.Nil, err
    }

    return userID, nil
}

func GetBearerToken(headers http.Header) (string, error) {
    authHead := headers.Get("Authorization")
    if authHead == "" {
        return "", errors.New("Authorization header is mising")
    }

    if !strings.HasPrefix(authHead, "Bearer ") {
        return "", errors.New("Token bearer not exists")
    }

    bearerToken := strings.TrimPrefix(authHead, "Bearer ")
    return bearerToken, nil
}

func MakeRefreshToken() (string, error) {
    key := make([]byte, 32)
    _, err := rand.Read(key)
    if err != nil {
        return "", err
    }

    hexStr := hex.EncodeToString(key)

    return hexStr, nil
}
