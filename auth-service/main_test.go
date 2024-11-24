package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// MockDatabase для имитации работы с базой данных
type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) CreateUser(user *User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockDatabase) FindUserByUsername(username string) (*User, error) {
	args := m.Called(username)
	return args.Get(0).(*User), args.Error(1)
}

func TestRegister(t *testing.T) {
	gin.SetMode(gin.TestMode) // Устанавливаем режим тестирования для gin

	r := gin.Default()
	mockDB := new(MockDatabase)
	db = mockDB

	r.POST("/register", register)

	t.Run("successful registration", func(t *testing.T) {
		user := &User{Username: "testuser", Password: "password"}
		mockDB.On("CreateUser", mock.AnythingOfType("*main.User")).Return(nil)

		jsonUser, _ := json.Marshal(user)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonUser))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"message": "Registration successful"}`, resp.Body.String())
		mockDB.AssertExpectations(t)
	})

	t.Run("register with existing username", func(t *testing.T) {
		user := &User{Username: "testuser", Password: "password"}
		mockDB.On("CreateUser", mock.AnythingOfType("*main.User")).Return(gorm.ErrDuplicatedKey)

		jsonUser, _ := json.Marshal(user)
		req, _ := http.NewRequest(http.MethodPost, "/register", bytes.NewBuffer(jsonUser))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"error": "Username already exists"}`, resp.Body.String())
		mockDB.AssertExpectations(t)
	})
}

func TestLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	mockDB := new(MockDatabase)
	db = mockDB

	r.POST("/login", login)

	t.Run("successful login", func(t *testing.T) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		user := &User{Username: "testuser", Password: string(hashedPassword)}
		mockDB.On("FindUserByUsername", "testuser").Return(user, nil)

		loginCredentials := &User{Username: "testuser", Password: "password"}
		jsonCredentials, _ := json.Marshal(loginCredentials)
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonCredentials))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		var response map[string]string
		json.Unmarshal(resp.Body.Bytes(), &response)
		assert.Contains(t, response, "token")
		mockDB.AssertExpectations(t)
	})

	t.Run("login with invalid credentials", func(t *testing.T) {
		user := &User{Username: "testuser", Password: "wronghash"}
		mockDB.On("FindUserByUsername", "testuser").Return(user, nil)

		loginCredentials := &User{Username: "testuser", Password: "wrongpassword"}
		jsonCredentials, _ := json.Marshal(loginCredentials)
		req, _ := http.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(jsonCredentials))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.JSONEq(t, `{"error": "Invalid credentials"}`, resp.Body.String())
		mockDB.AssertExpectations(t)
	})
}
