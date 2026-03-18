package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const kisBaseURL = "https://openapi.koreainvestment.com:9443"

type KISAuth struct {
	// appKey, appSecret, and baseURL are immutable after construction — safe to read without locking.
	appKey      string
	appSecret   string
	baseURL     string
	token       string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

func NewKISAuth(appKey, appSecret string) *KISAuth {
	return &KISAuth{appKey: appKey, appSecret: appSecret, baseURL: kisBaseURL}
}

// NewKISAuthWithBaseURL is for testing with a mock server URL.
func NewKISAuthWithBaseURL(appKey, appSecret, baseURL string) *KISAuth {
	return &KISAuth{appKey: appKey, appSecret: appSecret, baseURL: baseURL}
}

// SetTokenForTest pre-sets token state for unit tests.
func (a *KISAuth) SetTokenForTest(token string, expiry time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.token = token
	a.tokenExpiry = expiry
}

// GetToken returns a valid token, renewing if expiry is within 5 minutes.
func (a *KISAuth) GetToken(ctx context.Context) (string, error) {
	if a.appKey == "" || a.appSecret == "" {
		return "", fmt.Errorf("KIS_APP_KEY and KIS_APP_SECRET are required")
	}
	a.mu.RLock()
	if a.token != "" && time.Until(a.tokenExpiry) > 5*time.Minute {
		tok := a.token
		a.mu.RUnlock()
		return tok, nil
	}
	a.mu.RUnlock()
	return a.fetchToken(ctx)
}

// ForceRenewToken unconditionally fetches a new token (used after 401).
func (a *KISAuth) ForceRenewToken(ctx context.Context) error {
	a.mu.Lock()
	a.token = ""
	a.mu.Unlock()
	_, err := a.fetchToken(ctx)
	return err
}

func (a *KISAuth) fetchToken(ctx context.Context) (string, error) {
	body, err := json.Marshal(map[string]string{
		"grant_type": "client_credentials",
		"appkey":     a.appKey,
		"appsecret":  a.appSecret,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/oauth2/tokenP", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch KIS token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("KIS token endpoint returned HTTP %d", resp.StatusCode)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("empty access_token from KIS")
	}

	a.mu.Lock()
	a.token = result.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	a.mu.Unlock()

	return result.AccessToken, nil
}

// GetWSApprovalKey fetches a fresh WebSocket approval key.
func (a *KISAuth) GetWSApprovalKey(ctx context.Context) (string, error) {
	body, err := json.Marshal(map[string]string{
		"grant_type": "client_credentials",
		"appkey":     a.appKey,
		"secretkey":  a.appSecret,
	})
	if err != nil {
		return "", fmt.Errorf("marshal request body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/oauth2/Approval", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build WS approval request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch WS approval key: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ApprovalKey string `json:"approval_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode approval key response (HTTP %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("KIS WS approval endpoint returned HTTP %d (body: %+v)", resp.StatusCode, result)
	}
	if result.ApprovalKey == "" {
		return "", fmt.Errorf("empty approval_key from KIS")
	}
	return result.ApprovalKey, nil
}
