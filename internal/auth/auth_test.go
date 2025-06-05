package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	hash, _ := HashPassword("thisisapassword")
	err := CheckPassword(hash, "thisisapassword")
	if err != nil {
		t.Error("Expected: password and hash to match. Actual: password and hash did not match.")
	}
	err = CheckPassword(hash, "test")
	if err == nil {
		t.Error("Expected: password and hash not to match. Actual: password and hash matched.")
	}
}

func TestJWT(t *testing.T) {
	userId := uuid.New()
	val, err := MakeJWT(userId, "testsecret", time.Second*1)
	if err != nil {
		t.Errorf("Error when creating JWT: %v", err)
	}
	if len(val) == 0 {
		t.Errorf("JWT is empty: %v", err)
	}

	userId2, err := ValidateJWT(val, "testsecret")
	if err != nil {
		t.Errorf("Error when validating JWT: %v", err)
	}
	if userId != userId2 {
		t.Errorf("Expected: userId = %v. Actual: userId = %v", userId, userId2)
	}

	time.Sleep(time.Second * 2)
	_, err = ValidateJWT(val, "testsecret")
	if err == nil {
		t.Error("Token should have timed out")
	}
}

func TestGetBearerToken(t *testing.T) {
	header := http.Header{}
	header.Add("Authorization", "")
	val, err := GetBearerToken(header)
	if err == nil && len(val) == 0 {
		t.Error("Expected error, but did not get one")
	}
	header.Set("Authorization", "Bearer 123")
	val, err = GetBearerToken(header)
	if err != nil || val != "123" {
		t.Errorf("Expected token = '123', Actual: token = '%v'", val)
	}
}

func TestMakeRefreshToken(t *testing.T) {
	token, err := MakeRefreshToken()
	if err != nil {
		t.Errorf("error when generating random refresh token: %v", err)
	}
	fmt.Printf("token: %v\n", token)
}
