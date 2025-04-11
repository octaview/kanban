package auth

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte(os.Getenv("JWT_SECRET"))

func GenerateToken(userID string) (string, error) {
	expiryHours, _ := strconv.Atoi(os.Getenv("JWT_EXPIRY_HOURS"))
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Duration(expiryHours) * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["user_id"] == nil {
		return "", errors.New("invalid claims")
	}

	return claims["user_id"].(string), nil
}
