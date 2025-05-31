package auth

import (
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
	val, err := MakeJWT(userId, "testsecret", time.Second*2)
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

	time.Sleep(time.Second * 3)
	_, err = ValidateJWT(val, "testsecret")
	if err == nil {
		t.Error("Token should have timed out")
	}
}
