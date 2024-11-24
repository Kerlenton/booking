package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockDatabase struct {
	mock.Mock
}

func (m *MockDatabase) AutoMigrate(models ...interface{}) error {
	return m.Called(models).Error(0)
}

func (m *MockDatabase) Create(value interface{}) error {
	args := m.Called(value)
	return args.Error(0)
}

func (m *MockDatabase) Where(query interface{}, args ...interface{}) Database {
	args := m.Called(query, args)
	return m
}

func (m *MockDatabase) First(dest interface{}, conds ...interface{}) error {
	args := m.Called(dest, conds)
	return args.Error(0)
}

func (m *MockDatabase) Find(dest interface{}, conds ...interface{}) error {
	args := m.Called(dest, conds)
	return args.Error(0)
}

func TestCreateBooking(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	mockDB := new(MockDatabase)
	db = mockDB

	r.POST("/book", createBooking)

	t.Run("successfully create booking", func(t *testing.T) {
		booking := &Booking{RoomName: "Room1", StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Hour), UserID: 1}
		mockDB.On("Where", "room_name = ? AND start_time < ? AND end_time > ?", mock.Anything, mock.Anything, mock.Anything).Return(mockDB)
		mockDB.On("First", mock.Anything, mock.Anything).Return(gorm.ErrRecordNotFound)
		mockDB.On("Create", mock.AnythingOfType("*main.Booking")).Return(nil)

		claims := Claims{UserID: 1}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("secret"))

		jsonBooking, _ := json.Marshal(booking)
		req, _ := http.NewRequest(http.MethodPost, "/book", bytes.NewBuffer(jsonBooking))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"message": "Booking created successfully"}`, resp.Body.String())
		mockDB.AssertExpectations(t)
	})

	t.Run("booking conflict", func(t *testing.T) {
		booking := &Booking{RoomName: "Room1", StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Hour), UserID: 1}
		mockDB.On("Where", "room_name = ? AND start_time < ? AND end_time > ?", mock.Anything, mock.Anything, mock.Anything).Return(mockDB)
		mockDB.On("First", mock.Anything, mock.Anything).Return(nil) // Pretend we found a conflict

		claims := Claims{UserID: 1}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString([]byte("secret"))

		jsonBooking, _ := json.Marshal(booking)
		req, _ := http.NewRequest(http.MethodPost, "/book", bytes.NewBuffer(jsonBooking))
		req.Header.Set("Authorization", "Bearer "+tokenString)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.JSONEq(t, `{"error": "Room is already booked for this time"}`, resp.Body.String())
		mockDB.AssertExpectations(t)
	})

	t.Run("unauthorized request", func(t *testing.T) {
		booking := &Booking{RoomName: "Room1", StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Hour), UserID: 1}

		jsonBooking, _ := json.Marshal(booking)
		req, _ := http.NewRequest(http.MethodPost, "/book", bytes.NewBuffer(jsonBooking))
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusUnauthorized, resp.Code)
		assert.JSONEq(t, `{"error": "Unauthorized"}`, resp.Body.String())
	})
}

func TestGetBookings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	mockDB := new(MockDatabase)
	db = mockDB

	r.GET("/bookings", getBookings)

	t.Run("fetch bookings", func(t *testing.T) {
		booking := Booking{RoomName: "Room1", StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Hour), UserID: 1}
		mockDB.On("Find", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(0).(*[]Booking)
			*arg = append(*arg, booking)
		})

		req, _ := http.NewRequest(http.MethodGet, "/bookings", nil)
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		expectedResponse := gin.H{"bookings": []Booking{booking}}
		responseJSON, _ := json.Marshal(expectedResponse)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, string(responseJSON), resp.Body.String())
		mockDB.AssertExpectations(t)
	})
}
