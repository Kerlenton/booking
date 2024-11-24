package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	loginURL          = "http://localhost:8081/login"
	requestURLBooking = "http://localhost:8082/book"
	concurrency       = 50
	totalRequests     = 1000
)

type LoginResponse struct {
	Token string `json:"token"`
}

func sendRequest(method, url string, body []byte, headers map[string]string) (int, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func login() (string, error) {
	loginData := map[string]string{"username": "semyon", "password": "1234"}
	body, _ := json.Marshal(loginData)
	headers := map[string]string{"Content-Type": "application/json"}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodPost, loginURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed, status code: %d", resp.StatusCode)
	}

	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	return loginResp.Token, nil
}

func loadTestBooking(wg *sync.WaitGroup, token string) {
	defer wg.Done()
	headers := map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + token}
	bookingBody := []byte(`{"room_name": "Conference Room", "start_time": "2023-12-10T10:00:00Z", "end_time": "2023-12-10T11:00:00Z"}`)

	for i := 0; i < totalRequests/concurrency; i++ {
		statusCode, err := sendRequest(http.MethodPost, requestURLBooking, bookingBody, headers)
		if err != nil || statusCode != http.StatusOK {
			fmt.Printf("Error when sending booking: %v, status code: %d\n", err, statusCode)
		} else {
			fmt.Println("Booking created successfully, status code:", statusCode)
		}
	}
}

func main() {
	token, err := login()
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go loadTestBooking(&wg, token)
	}

	wg.Wait()
	fmt.Println("Load test completed")
}
