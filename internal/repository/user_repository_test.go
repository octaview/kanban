package repository_test

import (
	"context"
	"testing"

	"kanban/internal/model"
	"kanban/internal/repository"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	
	dialector := postgres.New(postgres.Config{
		DSN:                  "sqlmock_db_0",
		DriverName:           "postgres",
		Conn:                 db,
		PreferSimpleProtocol: true,
	})
	
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	assert.NoError(t, err)
	
	return gormDB, mock
}

func TestUserRepository_Create(t *testing.T) {
	// Arrange
	gormDB, mock := setupMockDB(t)
	userRepo := repository.NewUserRepository(gormDB)
	
	userID := uuid.New()
	user := &model.User{
		ID:             userID,
		Email:          "test@example.com",
		HashedPassword: "hashed_password",
		Name:           "Test User",
	}
	
	// Ожидаем SQL запрос на создание пользователя
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "users"`).
		WithArgs(sqlmock.AnyArg(), user.Email, user.HashedPassword, user.Name, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID.String()))
	mock.ExpectCommit()
	
	// Act
	err := userRepo.Create(context.Background(), user)
	
	// Assert
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail_Found(t *testing.T) {
	// Arrange
	gormDB, mock := setupMockDB(t)
	userRepo := repository.NewUserRepository(gormDB)
	
	userID := uuid.New()
	email := "test@example.com"
	
	// Ожидаем SQL запрос на поиск пользователя по email
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = .* LIMIT 1`).
		WithArgs(email).
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "hashed_password", "name", "created_at"}).
			AddRow(userID.String(), email, "hashed_password", "Test User", "2023-01-01 00:00:00"))
	
	// Act
	user, err := userRepo.FindByEmail(context.Background(), email)
	
	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, userID, user.ID)
	assert.Equal(t, email, user.Email)
	assert.Equal(t, "Test User", user.Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail_NotFound(t *testing.T) {
	// Arrange
	gormDB, mock := setupMockDB(t)
	userRepo := repository.NewUserRepository(gormDB)
	
	email := "nonexistent@example.com"
	
	// Ожидаем SQL запрос на поиск пользователя по email - не найден
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = .* LIMIT 1`).
		WithArgs(email).
		WillReturnError(gorm.ErrRecordNotFound)
	
	// Act
	user, err := userRepo.FindByEmail(context.Background(), email)
	
	// Assert
	assert.NoError(t, err) // Метод не возвращает ошибку при отсутствии записи
	assert.Nil(t, user)    // Но возвращает nil пользователя
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_FindByEmail_Error(t *testing.T) {
	// Arrange
	gormDB, mock := setupMockDB(t)
	userRepo := repository.NewUserRepository(gormDB)
	
	email := "test@example.com"
	
	// Ожидаем SQL запрос на поиск пользователя по email - произошла ошибка БД
	mock.ExpectQuery(`SELECT .* FROM "users" WHERE email = .* LIMIT 1`).
		WithArgs(email).
		WillReturnError(assert.AnError)
	
	// Act
	user, err := userRepo.FindByEmail(context.Background(), email)
	
	// Assert
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.NoError(t, mock.ExpectationsWereMet())
}