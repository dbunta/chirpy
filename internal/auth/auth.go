package auth

import (
	"fmt"
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

// not sure how i feel about this keyboard but i'll give it a chance
// it's not as rattly as I remember, I wonder if it was lubed.
// i think it may have been lubed at some point
// i think at some point it may have been lubed 9999999999999999999999
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	//var claims jwt.RegisteredClaims
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		return uuid.New(), fmt.Errorf("ValidateJWT Error parsing token: %w", err)
	}
	id, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.New(), fmt.Errorf("Validate JWT Error getting subject from token claims: %w", err)
	}
	return uuid.Parse(id)
}
