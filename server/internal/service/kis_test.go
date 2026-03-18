package service_test

import (
	"testing"

	"github.com/shinyoung/investment/internal/service"
)

func TestStripKRXSuffix(t *testing.T) {
	cases := []struct{ in, want string }{
		{"005930.KS", "005930"},
		{"247540.KQ", "247540"},
		{"AAPL", "AAPL"},
		{"005930", "005930"},
	}
	for _, c := range cases {
		got := service.StripKRXSuffix(c.in)
		if got != c.want {
			t.Errorf("StripKRXSuffix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
