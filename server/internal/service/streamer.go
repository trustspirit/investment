package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/shinyoung/investment/internal/ws"
)

type PriceStreamer struct {
	hub          *ws.Hub
	stocks       *StockService
	kisWS        *KISWebSocket
	pollInterval time.Duration
}

func NewPriceStreamer(hub *ws.Hub, stocks *StockService, kisWS *KISWebSocket) *PriceStreamer {
	return &PriceStreamer{
		hub:          hub,
		stocks:       stocks,
		kisWS:        kisWS,
		pollInterval: 5 * time.Second,
	}
}

func (ps *PriceStreamer) Start(ctx context.Context) {
	// Route Korean symbol events to KIS WebSocket (only when KIS is configured)
	if ps.kisWS != nil {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case evt := <-ps.hub.SymbolEvents:
					if IsKoreanSymbol(evt.Symbol) {
						if evt.Subscribe {
							ps.kisWS.Subscribe(evt.Symbol)
						} else {
							ps.kisWS.Unsubscribe(evt.Symbol)
						}
					}
				}
			}
		}()
	}

	// Yahoo polling loop for non-Korean symbols only
	go func() {
		ticker := time.NewTicker(ps.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("price streamer stopped")
				return
			case <-ticker.C:
				for _, symbol := range ps.hub.SubscribedSymbols() {
					if IsKoreanSymbol(symbol) {
						continue // handled by KIS WebSocket push
					}
					quote, err := ps.stocks.GetQuote(ctx, symbol)
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
