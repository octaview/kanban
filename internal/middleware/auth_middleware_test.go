package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kanban/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()

	jwtSecret := "test-secret-key"

	// Защищенный маршрут
	protected := r.Group("/protected")

	// Добавляем middleware аутентификации
	protected.Use(middleware.JWTAuthMiddleware(jwtSecret))

	// Обработчик для проверки middleware
	protected.GET("/resource", func(c *gin.Context) {
		// Получаем userID из контекста
		userID, exists := c.Get(middleware.UserIDKey)
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "User ID not found in context"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Access granted",
			"user_id": userID,
		})
	})

	return r
}

func generateTestToken(userID uuid.UUID, jwtSecret string) string {
	claims := jwt.MapClaims{
		"user_id": userID.String(),
		"exp":     jwt.NewNumericDate(time.Now().Add(time.Hour * 24)), // Set expiration to 24 hours from now
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte(jwtSecret))

	return tokenString
}

func TestJWTAuthMiddleware_ValidToken(t *testing.T) {
	// Arrange
	router := setupRouter()
	userID := uuid.New()
	token := generateTestToken(userID, "test-secret-key")

	// Создаем запрос с валидным токеном
	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusOK, resp.Code)

	// Проверяем успешный доступ и соответствие ID пользователя
	assert.Contains(t, resp.Body.String(), "Access granted")
	assert.Contains(t, resp.Body.String(), userID.String())
}

func TestJWTAuthMiddleware_NoAuthHeader(t *testing.T) {
	// Arrange
	router := setupRouter()

	// Создаем запрос без заголовка авторизации
	req, _ := http.NewRequest("GET", "/protected/resource", nil)

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Authorization header is required")
}

func TestJWTAuthMiddleware_InvalidAuthFormat(t *testing.T) {
	// Arrange
	router := setupRouter()

	// Создаем запрос с неверным форматом заголовка
	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "InvalidFormat token123")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Authorization header format must be Bearer {token}")
}

func TestJWTAuthMiddleware_InvalidToken(t *testing.T) {
	// Arrange
	router := setupRouter()

	// Создаем запрос с недействительным токеном
	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid or expired token")
}

func TestJWTAuthMiddleware_TokenWithInvalidUserID(t *testing.T) {
	// Arrange
	router := setupRouter()

	// Создаем токен с недействительным форматом ID пользователя
	claims := jwt.MapClaims{
		"user_id": "not-a-valid-uuid",
		"exp":     jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret-key"))

	// Создаем запрос с токеном, содержащим неверный формат ID
	req, _ := http.NewRequest("GET", "/protected/resource", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, resp.Code)
	assert.Contains(t, resp.Body.String(), "Invalid user ID in token")
}
