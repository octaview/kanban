package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"kanban/internal/handler"
	"kanban/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

// Мок репозитория пользователей
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *model.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	args := m.Called(ctx, email)
	user := args.Get(0)
	if user == nil {
		return nil, args.Error(1)
	}
	return user.(*model.User), args.Error(1)
}

func setupTest() (*gin.Engine, *MockUserRepository) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	mockRepo := new(MockUserRepository)
	userHandler := handler.NewUserHandler(mockRepo)

	r.POST("/register", userHandler.Register)
	r.POST("/login", userHandler.Login)

	// Устанавливаем JWT_SECRET для тестов
	os.Setenv("JWT_SECRET", "test-secret")
	return r, mockRepo
}

func TestRegister_Success(t *testing.T) {
	// Arrange
	router, mockRepo := setupTest()

	// Мокаем методы репозитория
	mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(nil, nil)
	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	// Создаем тестовый запрос
	reqBody := handler.RegisterRequest{
		Name:     "Test User",
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusCreated, resp.Code)

	var response handler.AuthResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, reqBody.Name, response.User.Name)
	assert.Equal(t, reqBody.Email, response.User.Email)

	mockRepo.AssertExpectations(t)
}

func TestRegister_UserAlreadyExists(t *testing.T) {
	// Arrange
	router, mockRepo := setupTest()

	// Мокаем методы репозитория - пользователь уже существует
	existingUser := &model.User{
		ID:             uuid.New(),
		Email:          "existing@example.com",
		HashedPassword: "hashed_password",
		Name:           "Existing User",
	}
	mockRepo.On("FindByEmail", mock.Anything, "existing@example.com").Return(existingUser, nil)

	// Создаем тестовый запрос
	reqBody := handler.RegisterRequest{
		Name:     "Test User",
		Email:    "existing@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/register", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusConflict, resp.Code)

	var response map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "User with this email already exists", response["error"])

	mockRepo.AssertExpectations(t)
}

func TestLogin_Success(t *testing.T) {
	// Arrange
	router, mockRepo := setupTest()

	// Создаем хешированный пароль для тестового пользователя
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	testUser := &model.User{
		ID:             uuid.New(),
		Email:          "test@example.com",
		HashedPassword: string(hashedPassword),
		Name:           "Test User",
	}

	// Мокаем метод репозитория
	mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(testUser, nil)

	// Создаем тестовый запрос
	reqBody := handler.LoginRequest{
		Email:    "test@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusOK, resp.Code)

	var response handler.AuthResponse
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.Token)
	assert.Equal(t, testUser.Name, response.User.Name)
	assert.Equal(t, testUser.Email, response.User.Email)
	assert.Equal(t, testUser.ID.String(), response.User.ID)

	mockRepo.AssertExpectations(t)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	// Arrange
	router, mockRepo := setupTest()

	// Создаем хешированный пароль для тестового пользователя
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("correct_password"), bcrypt.DefaultCost)
	testUser := &model.User{
		ID:             uuid.New(),
		Email:          "test@example.com",
		HashedPassword: string(hashedPassword),
		Name:           "Test User",
	}

	// Мокаем метод репозитория
	mockRepo.On("FindByEmail", mock.Anything, "test@example.com").Return(testUser, nil)

	// Создаем тестовый запрос с неверным паролем
	reqBody := handler.LoginRequest{
		Email:    "test@example.com",
		Password: "wrong_password",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	var response map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid credentials", response["error"])

	mockRepo.AssertExpectations(t)
}

func TestLogin_UserNotFound(t *testing.T) {
	// Arrange
	router, mockRepo := setupTest()

	// Мокаем метод репозитория - пользователь не найден
	mockRepo.On("FindByEmail", mock.Anything, "nonexistent@example.com").Return(nil, nil)

	// Создаем тестовый запрос
	reqBody := handler.LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	// Act
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, resp.Code)

	var response map[string]string
	err := json.Unmarshal(resp.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Invalid credentials", response["error"])

	mockRepo.AssertExpectations(t)
}
