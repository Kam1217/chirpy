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

func TestValidateJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "superSecretKey"
	wrongSecret := "anotherSecret"

	t.Run("Valid Token", func(t *testing.T) {
		validToken, err := MakeJWT(userID, tokenSecret, time.Minute)
		if err != nil {
			t.Fatalf("Failed to make valid token for test: %v", err)
		}
		validatedID, err := ValidateJWT(validToken, tokenSecret)
		if err != nil {
			t.Errorf("Expected no error for valid token, got: %v", err)
		}
		if validatedID != userID {
			t.Errorf("Expected userID %s, got %s", userID, validatedID)
		}
	})

	t.Run("Invalid Secret", func(t *testing.T) {
		validToken, err := MakeJWT(userID, tokenSecret, time.Minute)
		if err != nil {
			t.Fatalf("Failed to make valid token for test: %v", err)
		}
		_, err = ValidateJWT(validToken, wrongSecret)
		if err == nil {
			t.Error("Expected error for invalid secret, got none")
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		expiredToken, err := MakeJWT(userID, tokenSecret, -time.Hour)
		if err != nil {
			t.Fatalf("Failed to make expired token for test: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
		_, err = ValidateJWT(expiredToken, tokenSecret)
		if err == nil {
			t.Error("Expected error for expired token, got none")
		}

	})

	t.Run("Malformed Token", func(t *testing.T) {
		malformedToken := "this.is.not.a.jwt"
		_, err := ValidateJWT(malformedToken, tokenSecret)
		if err == nil {
			t.Error("Expected error for malformed token, got none")
		}
	})
}
