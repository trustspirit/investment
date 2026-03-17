package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"

	"github.com/shinyoung/investment/internal/model"
)

type InsightScheduler struct {
	cache     sync.Map
	ai        AIProvider
	yahoo     *YahooService
	news      *NewsService
	watchlist *WatchlistService
	scheduler gocron.Scheduler
}

func NewInsightScheduler(ai AIProvider, yahoo *YahooService, news *NewsService, watchlist *WatchlistService) *InsightScheduler {
	return &InsightScheduler{
		ai:        ai,
		yahoo:     yahoo,
		news:      news,
		watchlist: watchlist,
	}
}

func (is *InsightScheduler) Start(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return err
	}
	is.scheduler = scheduler

	_, err = scheduler.NewJob(
		gocron.DurationJob(30*time.Minute),
		gocron.NewTask(func() {
			is.refreshAll(ctx)
		}),
	)
	if err != nil {
		return err
	}

	scheduler.Start()
	slog.Info("insight scheduler started", "interval", "30m")
	return nil
}

func (is *InsightScheduler) Stop() {
	if is.scheduler != nil {
		if err := is.scheduler.Shutdown(); err != nil {
			slog.Error("failed to stop insight scheduler", "error", err)
		}
	}
}

func (is *InsightScheduler) GetCached(symbol string) (model.AIInsight, bool) {
	v, ok := is.cache.Load(symbol)
	if !ok {
		return model.AIInsight{}, false
	}
	return v.(model.AIInsight), true
}

func (is *InsightScheduler) GenerateForSymbol(ctx context.Context, symbol string) (model.AIInsight, error) {
	insight, err := is.generateInsight(ctx, symbol)
	if err != nil {
		return model.AIInsight{}, err
	}
	is.cache.Store(symbol, insight)
	return insight, nil
}

func (is *InsightScheduler) refreshAll(ctx context.Context) {
	symbols := is.watchlist.Symbols()
	slog.Info("refreshing insights for watchlist", "count", len(symbols))

	for _, symbol := range symbols {
		insight, err := is.generateInsight(ctx, symbol)
		if err != nil {
			slog.Error("failed to refresh insight", "symbol", symbol, "error", err)
			continue
		}
		is.cache.Store(symbol, insight)
		slog.Info("refreshed insight", "symbol", symbol)
	}
}

func (is *InsightScheduler) GenerateStrategyForSymbol(ctx context.Context, symbol string) (model.AITradeStrategy, error) {
	quote, err := is.yahoo.GetQuote(ctx, symbol)
	if err != nil {
		return model.AITradeStrategy{}, err
	}

	info, err := is.yahoo.GetCompanyInfo(ctx, symbol)
	if err != nil {
		slog.Warn("failed to get company info for strategy", "symbol", symbol, "error", err)
		info = model.CompanyInfo{Symbol: symbol, Name: quote.Name}
	}

	companyNews, err := is.yahoo.GetNews(ctx, symbol)
	if err != nil {
		slog.Warn("failed to get news for strategy", "symbol", symbol, "error", err)
		companyNews = nil
	}

	broadNews := is.news.GetBroadNews(ctx, symbol, info.Sector, info.Industry)

	var recommendation *model.RecommendationData
	rec, err := is.yahoo.GetRecommendation(ctx, symbol)
	if err != nil {
		slog.Warn("failed to get recommendation for strategy", "symbol", symbol, "error", err)
	} else {
		recommendation = &rec
	}

	history, err := is.yahoo.GetHistoricalData(ctx, symbol, "1mo")
	if err != nil {
		slog.Warn("failed to get historical data for strategy", "symbol", symbol, "error", err)
		history = nil
	}

	return is.ai.GenerateStrategy(ctx, symbol, info, companyNews, broadNews, quote, recommendation, history)
}

func (is *InsightScheduler) generateInsight(ctx context.Context, symbol string) (model.AIInsight, error) {
	quote, err := is.yahoo.GetQuote(ctx, symbol)
	if err != nil {
		return model.AIInsight{}, err
	}

	info, err := is.yahoo.GetCompanyInfo(ctx, symbol)
	if err != nil {
		slog.Warn("failed to get company info for insight", "symbol", symbol, "error", err)
		info = model.CompanyInfo{Symbol: symbol, Name: quote.Name}
	}

	companyNews, err := is.yahoo.GetNews(ctx, symbol)
	if err != nil {
		slog.Warn("failed to get news for insight", "symbol", symbol, "error", err)
		companyNews = nil
	}

	broadNews := is.news.GetBroadNews(ctx, symbol, info.Sector, info.Industry)

	var recommendation *model.RecommendationData
	rec, err := is.yahoo.GetRecommendation(ctx, symbol)
	if err != nil {
		slog.Warn("failed to get recommendation for insight", "symbol", symbol, "error", err)
	} else {
		recommendation = &rec
	}

	return is.ai.GenerateInsight(ctx, symbol, info, companyNews, broadNews, quote, recommendation)
}
