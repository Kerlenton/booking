package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSendNotification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Устанавливаем роутинг
	r := gin.Default()
	r.POST("/notify", sendNotification)

	t.Run("successfully send notification", func(t *testing.T) {
		notification := `{"message": "Test Notification"}`
		req, _ := http.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(notification))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusOK, resp.Code)
		assert.JSONEq(t, `{"message": "Notification sent"}`, resp.Body.String())
	})

	t.Run("bad request with invalid JSON", func(t *testing.T) {
		invalidJSON := `{"message": "Test Notification"`
		req, _ := http.NewRequest(http.MethodPost, "/notify", bytes.NewBufferString(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()

		r.ServeHTTP(resp, req)

		assert.Equal(t, http.StatusBadRequest, resp.Code)
		assert.Contains(t, resp.Body.String(), "error")
	})
}
