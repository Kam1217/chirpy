package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMakeJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "superSecretKey"
	expiresIn := time.Hour

	tokenString, err := MakeJWT(userID, tokenSecret, expiresIn)
	if err != nil {
		t.Fatalf("MakeJWT returned an error: %v", err)
	}

	if tokenString == "" {
		t.Error("MakeJWT returned an empty token string")
	}

	validatedUserID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Errorf("ValidateJWT failed to validate token created by MakeJWT: %v", err)
	}

	if validatedUserID != userID {
		t.Errorf("Validated UserID mismatch: got %s, want %s", validatedUserID, userID)
	}
}
