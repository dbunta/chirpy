package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	pwBytes, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	if err != nil {
		return "", fmt.Errorf("there was an issue hashing the password: %w", err)
	}
	return string(pwBytes), nil
}

func CheckPassword(hash, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		return fmt.Errorf("incorrect password")
	}
	return nil
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
		Subject:   userID.String(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", fmt.Errorf("MakeJWT Error getting signed string: %w", err)
	}
	return tokenString, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	//var claims jwt.RegisteredClaims
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.New(), fmt.Errorf("validateJWT error parsing token: %w", err)
	}
	id, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.New(), fmt.Errorf("validate jwt error getting subject from token claims: %w", err)
	}
	return uuid.Parse(id)
}

func GetBearerToken(header http.Header) (string, error) {
	authHeader := header.Get("Authorization")
	if len(authHeader) == 0 {
		return "", fmt.Errorf("authorization header missing")
	}
	token, found := strings.CutPrefix(authHeader, "Bearer ")
	if !found {
		return "", fmt.Errorf("authorization header in unexpected format")
	}
	return token, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return "", fmt.Errorf("error generating refresh token: %v", err)
	}
	retval := hex.EncodeToString(key)

	return retval, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	apiKey := headers.Get("Authorization")
	apiKey, found := strings.CutPrefix(apiKey, "ApiKey ")
	if !found {
		return "", fmt.Errorf("api key not found")
	}
	return apiKey, nil
}
