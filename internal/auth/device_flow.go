package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

func Login(baseURL string) (*Credentials, error) {
	resp, err := http.Post(baseURL+"/oauth/device", "application/json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device flow: %w", err)
	}
	defer resp.Body.Close()

	var dc deviceCodeResponse
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Printf("\nOpen this URL in your browser to sign in:\n\n  %s\n\n", dc.VerificationURIComplete)
	fmt.Printf("Or visit %s and enter code: %s\n\n", dc.VerificationURI, dc.UserCode)

	openBrowser(dc.VerificationURIComplete)
	fmt.Println("Waiting for authorization...")

	interval := dc.Interval
	if interval == 0 {
		interval = 5
	}

	for {
		time.Sleep(time.Duration(interval) * time.Second)

		token, err := pollToken(baseURL, dc.DeviceCode)
		if err != nil {
			return nil, err
		}

		switch token.Error {
		case "authorization_pending":
			continue
		case "":
			expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
			return &Credentials{
				AccessToken:  token.AccessToken,
				RefreshToken: token.RefreshToken,
				ExpiresAt:    expiresAt,
			}, nil
		default:
			return nil, fmt.Errorf("authorization failed: %s", token.Error)
		}
	}
}

func Refresh(baseURL, refreshToken string) (*Credentials, error) {
	body := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
	}

	jsonData, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+"/oauth/token", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if token.Error != "" {
		return nil, fmt.Errorf("refresh failed: %s", token.Error)
	}

	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	return &Credentials{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func pollToken(baseURL, deviceCode string) (*tokenResponse, error) {
	body := map[string]string{
		"grant_type":  "urn:ietf:params:oauth:grant-type:device_code",
		"device_code": deviceCode,
	}

	jsonData, _ := json.Marshal(body)
	resp, err := http.Post(baseURL+"/oauth/token", "application/json", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to poll token: %w", err)
	}
	defer resp.Body.Close()

	data, _ := io.ReadAll(resp.Body)

	var token tokenResponse
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	return &token, nil
}

func openBrowser(url string) {
	var cmd string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "linux":
		cmd = "xdg-open"
	default:
		return
	}
	exec.Command(cmd, url).Start() //nolint:errcheck
}
