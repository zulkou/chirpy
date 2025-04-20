package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHash(t *testing.T) {
    password := "VerySecret"
    hashed, err := HashPassword(password)
    if err != nil {
        t.Errorf("Hashing error: %v", err)
    }
    err = CheckPasswordHash(hashed, password)
    if err != nil {
        t.Errorf("Validate hash error: %v", err)
    }
}

func TestJWT(t *testing.T) {
    id, _ := uuid.NewRandom()
    secret := "ASecretString"
    expires := 24 * time.Hour
    token, err := MakeJWT(id, secret, expires)
    if err != nil {
        t.Errorf("Failed to create token: %v", err)
    }

    procID, err := ValidateJWT(token, secret)
    if err != nil {
        t.Errorf("Failed to validate JWT: %v", err)
    }

    if procID != id {
        t.Errorf("JWT changes ID from %v to %v", id, procID)
    }
}
