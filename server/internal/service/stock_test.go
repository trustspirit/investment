package service_test

import (
	"testing"

	"github.com/shinyoung/investment/internal/service"
)

func TestIsKoreanSymbol(t *testing.T) {
	cases := []struct {
		symbol string
		want   bool
	}{
		{"005930.KS", true},
		{"247540.KQ", true},
		{"AAPL", false},
		{"TSLA", false},
		{"^KS11", false},
		{"005930", false},
		{"", false},
	}
	for _, c := range cases {
		got := service.IsKoreanSymbol(c.symbol)
		if got != c.want {
			t.Errorf("IsKoreanSymbol(%q) = %v, want %v", c.symbol, got, c.want)
		}
	}
}

func TestNewStockService_DoesNotPanic(t *testing.T) {
	// StockService with nil kis must not panic on creation
	stocks := service.NewStockService(nil, nil)
	if stocks == nil {
		t.Fatal("expected non-nil StockService")
	}
}
