package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shinyoung/investment/internal/service"
)

func TestKISAuth_GetToken_FetchesOnFirstCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok1", "expires_in": 86400})
	}))
	defer srv.Close()

	auth := service.NewKISAuthWithBaseURL("key", "secret", srv.URL)
	tok, err := auth.GetToken(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if tok != "tok1" {
		t.Fatalf("want tok1 got %s", tok)
	}
	if callCount != 1 {
		t.Fatalf("want 1 call got %d", callCount)
	}
}

func TestKISAuth_GetToken_UsesCacheIfFresh(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok1", "expires_in": 86400})
	}))
	defer srv.Close()

	auth := service.NewKISAuthWithBaseURL("key", "secret", srv.URL)
	auth.GetToken(context.Background())
	auth.GetToken(context.Background())
	if callCount != 1 {
		t.Fatalf("want 1 fetch (cached) got %d", callCount)
	}
}

func TestKISAuth_ForceRenewToken_AlwaysFetches(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok-new", "expires_in": 86400})
	}))
	defer srv.Close()

	auth := service.NewKISAuthWithBaseURL("key", "secret", srv.URL)
	auth.GetToken(context.Background())
	if err := auth.ForceRenewToken(context.Background()); err != nil {
		t.Fatal(err)
	}
	if callCount != 2 {
		t.Fatalf("want 2 fetches got %d", callCount)
	}
}

func TestKISAuth_GetToken_RenewsWhenNearExpiry(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "expires_in": 86400})
	}))
	defer srv.Close()

	auth := service.NewKISAuthWithBaseURL("key", "secret", srv.URL)
	// Pre-set token expiring in 3 minutes (within 5-min renewal window)
	auth.SetTokenForTest("old", time.Now().Add(3*time.Minute))

	auth.GetToken(context.Background())
	if callCount != 1 {
		t.Fatalf("want 1 renewal call got %d", callCount)
	}
}

func TestKISAuth_GetToken_ErrorsWithMissingCredentials(t *testing.T) {
	auth := service.NewKISAuth("", "")
	_, err := auth.GetToken(context.Background())
	if err == nil {
		t.Fatal("want error for missing credentials")
	}
}
