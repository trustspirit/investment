package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/shinyoung/investment/internal/ws"
)

type PriceStreamer struct {
	hub          *ws.Hub
	yahoo        *YahooService
	pollInterval time.Duration
}

func NewPriceStreamer(hub *ws.Hub, yahoo *YahooService) *PriceStreamer {
	return &PriceStreamer{
		hub:          hub,
		yahoo:        yahoo,
		pollInterval: 5 * time.Second,
	}
}

func (ps *PriceStreamer) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(ps.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("price streamer stopped")
				return
			case <-ticker.C:
				symbols := ps.hub.SubscribedSymbols()
				for _, symbol := range symbols {
					quote, err := ps.yahoo.GetQuote(ctx, symbol)
					if err != nil {
						slog.Error("failed to fetch quote for stream", "symbol", symbol, "error", err)
						continue
					}
					ps.hub.BroadcastPrice(symbol, quote)
				}
			}
		}
	}()

	slog.Info("price streamer started", "interval", ps.pollInterval)
}
