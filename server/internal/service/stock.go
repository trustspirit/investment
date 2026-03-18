package service

import (
	"context"
	"strings"

	"github.com/shinyoung/investment/internal/model"
)

// IsKoreanSymbol reports whether symbol is a KRX-listed stock.
func IsKoreanSymbol(symbol string) bool {
	return strings.HasSuffix(symbol, ".KS") || strings.HasSuffix(symbol, ".KQ")
}

type StockService struct {
	yahoo *YahooService
	kis   *KISService
}

func NewStockService(yahoo *YahooService, kis *KISService) *StockService {
	return &StockService{yahoo: yahoo, kis: kis}
}

func (s *StockService) GetQuote(ctx context.Context, symbol string) (model.StockQuote, error) {
	if IsKoreanSymbol(symbol) && s.kis != nil {
		return s.kis.GetQuote(ctx, symbol)
	}
	return s.yahoo.GetQuote(ctx, symbol)
}

func (s *StockService) GetHistoricalData(ctx context.Context, symbol string, chartRange string) ([]model.HistoricalDataPoint, error) {
	if IsKoreanSymbol(symbol) && s.kis != nil {
		// KIS for real-time intraday; Yahoo for historical ranges (same intervals as US stocks)
		switch chartRange {
		case "1d", "pre":
			return s.kis.GetHistoricalData(ctx, symbol, chartRange)
		default:
			return s.yahoo.GetHistoricalData(ctx, symbol, chartRange)
		}
	}
	return s.yahoo.GetHistoricalData(ctx, symbol, chartRange)
}

func (s *StockService) GetCompanyInfo(ctx context.Context, symbol string) (model.CompanyInfo, error) {
	if IsKoreanSymbol(symbol) && s.kis != nil {
		return s.kis.GetCompanyInfo(ctx, symbol)
	}
	return s.yahoo.GetCompanyInfo(ctx, symbol)
}

func (s *StockService) GetNews(ctx context.Context, symbol string) ([]model.NewsArticle, error) {
	// News always via Yahoo (KIS has no news API)
	return s.yahoo.GetNews(ctx, symbol)
}

func (s *StockService) GetRecommendation(ctx context.Context, symbol string) (model.RecommendationData, error) {
	// Analyst consensus always via Yahoo
	return s.yahoo.GetRecommendation(ctx, symbol)
}

func (s *StockService) GetMarketIndicators(ctx context.Context) ([]model.MarketIndicator, error) {
	// KOSPI/KOSDAQ index symbols (^KS11, ^KQ11) are not .KS/.KQ — Yahoo only
	return s.yahoo.GetMarketIndicators(ctx)
}

func (s *StockService) SearchSymbol(ctx context.Context, query string) ([]model.SymbolSearchResult, error) {
	return s.yahoo.SearchSymbol(ctx, query)
}
