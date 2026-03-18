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
	finnhubWS    *FinnhubWebSocket
	pollInterval time.Duration
}

func NewPriceStreamer(hub *ws.Hub, stocks *StockService, kisWS *KISWebSocket, finnhubWS *FinnhubWebSocket) *PriceStreamer {
	return &PriceStreamer{
		hub:          hub,
		stocks:       stocks,
		kisWS:        kisWS,
		finnhubWS:    finnhubWS,
		pollInterval: 5 * time.Second,
	}
}

func (ps *PriceStreamer) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-ps.hub.SymbolEvents:
				if IsKoreanSymbol(evt.Symbol) {
					if ps.kisWS != nil {
						if evt.Subscribe {
							ps.kisWS.Subscribe(evt.Symbol)
						} else {
							ps.kisWS.Unsubscribe(evt.Symbol)
						}
					}
				} else if ps.finnhubWS != nil {
					if evt.Subscribe {
						// Fetch previous close for change calculation before subscribing
						go func(sym string) {
							quote, err := ps.stocks.GetQuote(ctx, sym)
							if err == nil {
								ps.finnhubWS.SetPreviousClose(sym, quote.Price-quote.Change)
							}
							ps.finnhubWS.Subscribe(sym)
						}(evt.Symbol)
					} else {
						ps.finnhubWS.Unsubscribe(evt.Symbol)
					}
				}
			}
		}
	}()

	// Yahoo polling fallback — only for non-Korean symbols when Finnhub is not configured
	if ps.finnhubWS == nil {
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
							continue
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
	}

	slog.Info("price streamer started", "finnhub", ps.finnhubWS != nil, "interval", ps.pollInterval)
}
