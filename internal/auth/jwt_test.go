package auth_test

import (
	"os"
	"testing"
	"time"

	"kanban/internal/auth"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndParseToken(t *testing.T) {
	// Устанавливаем переменные окружения для тестов
	os.Setenv("JWT_SECRET", "test-secret-key")
	os.Setenv("JWT_EXPIRY_HOURS", "24")

	// Генерируем токен
	userID := "test-user-id"
	token, err := auth.GenerateToken(userID)
	
	// Проверяем, что токен создан без ошибок
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	
	// Парсим токен
	parsedUserID, err := auth.ParseToken(token)
	
	// Проверяем, что токен был успешно проверен и из него извлечен правильный ID пользователя
	assert.NoError(t, err)
	assert.Equal(t, userID, parsedUserID)
}

func TestParseToken_InvalidToken(t *testing.T) {
	// Устанавливаем переменные окружения для тестов
	os.Setenv("JWT_SECRET", "test-secret-key")
	
	// Пытаемся парсить неверный токен
	_, err := auth.ParseToken("invalid-token")
	
	// Проверяем, что возникла ошибка
	assert.Error(t, err)
	assert.Equal(t, "invalid token", err.Error())
}

func TestParseToken_ExpiredToken(t *testing.T) {
	// Устанавливаем переменные окружения для тестов
	os.Setenv("JWT_SECRET", "test-secret-key")
	
	// Создаем токен с истекшим сроком действия
	claims := jwt.MapClaims{
		"user_id": "test-user-id",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(), // Токен истек 1 час назад
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := token.SignedString([]byte("test-secret-key"))
	
	// Пытаемся парсить истекший токен
	_, err := auth.ParseToken(expiredToken)
	
	// Проверяем, что возникла ошибка
	assert.Error(t, err)
	assert.Equal(t, "invalid token", err.Error())
}

func TestParseToken_MissingClaims(t *testing.T) {
	// Устанавливаем переменные окружения для тестов
	os.Setenv("JWT_SECRET", "test-secret-key")
	
	// Создаем токен без ID пользователя
	claims := jwt.MapClaims{
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		// Отсутствует "user_id"
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenWithoutUserID, _ := token.SignedString([]byte("test-secret-key"))
	
	// Пытаемся парсить токен
	_, err := auth.ParseToken(tokenWithoutUserID)
	
	// Проверяем, что возникла ошибка
	assert.Error(t, err)
	assert.Equal(t, "invalid claims", err.Error())
}