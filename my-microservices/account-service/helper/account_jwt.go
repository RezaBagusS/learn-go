package helper

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateAccessToken(clientID string) (string, error) {
	var secretKey = []byte(os.Getenv("JWT_SECRET_KEY"))
	claims := jwt.MapClaims{
		"iss": "jwt-account-service",
		"sub": clientID,
		"jti": uuid.New().String(),
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(15 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}
