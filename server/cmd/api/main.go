package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/shinyoung/investment/internal/config"
	"github.com/shinyoung/investment/internal/handler"
	"github.com/shinyoung/investment/internal/middleware"
	"github.com/shinyoung/investment/internal/service"
	"github.com/shinyoung/investment/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	var aiProvider service.AIProvider
	apiKey, keyErr := selectAPIKey(cfg)
	if keyErr != nil {
		slog.Warn("AI provider not configured — AI insight endpoints will be unavailable", "reason", keyErr.Error())
		aiProvider = &service.NoopProvider{}
	} else {
		p, providerErr := service.NewAIProvider(cfg.AIProvider, apiKey, cfg.AIModel)
		if providerErr != nil {
			slog.Error("failed to create AI provider", "error", providerErr)
			os.Exit(1)
		}
		aiProvider = p
	}

	yahooService := service.NewYahooService()
	newsService := service.NewNewsService(yahooService, aiProvider)
	watchlistService := service.NewWatchlistService()
	insightScheduler := service.NewInsightScheduler(aiProvider, yahooService, newsService, watchlistService)

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := insightScheduler.Start(appCtx); err != nil {
		slog.Error("failed to start insight scheduler", "error", err)
		os.Exit(1)
	}
	defer insightScheduler.Stop()

	hub := ws.NewHub()
	go hub.Run()

	priceStreamer := service.NewPriceStreamer(hub, yahooService)
	priceStreamer.Start(appCtx)

	stockHandler := handler.NewStockHandler(yahooService, newsService)
	watchlistHandler := handler.NewWatchlistHandler(watchlistService)
	insightHandler := handler.NewInsightHandler(insightScheduler)

	router := chi.NewRouter()
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}).Handler)
	router.Use(middleware.Logger)

	router.Get("/api/stocks/search", stockHandler.Search)
	router.Get("/api/stocks/{symbol}/quote", stockHandler.GetQuote)
	router.Get("/api/stocks/{symbol}/chart", stockHandler.GetChart)
	router.Get("/api/stocks/{symbol}/info", stockHandler.GetInfo)
	router.Get("/api/stocks/{symbol}/news", stockHandler.GetNews)
	router.Get("/api/stocks/{symbol}/recommendation", stockHandler.GetRecommendation)

	router.Get("/api/watchlist", watchlistHandler.List)
	router.Post("/api/watchlist", watchlistHandler.Add)
	router.Delete("/api/watchlist/{symbol}", watchlistHandler.Remove)

	router.Get("/api/insights/{symbol}", insightHandler.Get)
	router.Post("/api/insights/{symbol}/generate", insightHandler.Generate)
	router.Post("/api/insights/{symbol}/strategy", insightHandler.GenerateStrategy)

	router.Get("/api/market/indicators", stockHandler.GetMarketIndicators)

	router.Get("/ws", handler.HandleWebSocket(hub))

	server := &http.Server{
		Addr:              cfg.Address(),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("starting API server", "addr", cfg.Address(), "aiProvider", cfg.AIProvider, "aiModel", cfg.AIModel)
		err := server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		slog.Error("server failed", "error", err)
		cancel()
		os.Exit(1)
	case sig := <-signalCh:
		slog.Info("received shutdown signal", "signal", sig.String())
	}

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
		if closeErr := server.Close(); closeErr != nil {
			slog.Error("failed to force-close server", "error", closeErr)
		}
	}

	slog.Info("server stopped")
}

func selectAPIKey(cfg config.Config) (string, error) {
	switch cfg.AIProvider {
	case config.ProviderAnthropic:
		if cfg.AnthropicAPIKey == "" {
			return "", errors.New("ANTHROPIC_API_KEY is required when AI_PROVIDER=anthropic")
		}
		return cfg.AnthropicAPIKey, nil
	case config.ProviderOpenAI:
		if cfg.OpenAIAPIKey == "" {
			return "", errors.New("OPENAI_API_KEY is required when AI_PROVIDER=openai")
		}
		return cfg.OpenAIAPIKey, nil
	default:
		return "", errors.New("unsupported AI provider")
	}
}
