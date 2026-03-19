package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shinyoung/investment/internal/model"
	ws "github.com/shinyoung/investment/internal/ws"
)

const finnhubWSURL = "wss://ws.finnhub.io"

type FinnhubWebSocket struct {
	apiKey     string
	hub        *ws.Hub
	conn       *websocket.Conn
	subscribed map[string]bool // symbol -> subscribed
	mu         sync.Mutex
	subCh      chan string
	unsubCh    chan string
	backoff    time.Duration
	// Track last known price to compute change from previous close
	prevClose map[string]float64
}

func NewFinnhubWebSocket(apiKey string, hub *ws.Hub) *FinnhubWebSocket {
	return &FinnhubWebSocket{
		apiKey:     apiKey,
		hub:        hub,
		subscribed: make(map[string]bool),
		subCh:      make(chan string, 32),
		unsubCh:    make(chan string, 32),
		backoff:    1 * time.Second,
		prevClose:  make(map[string]float64),
	}
}

func (f *FinnhubWebSocket) Start(ctx context.Context) {
	go f.runLoop(ctx)
}

func (f *FinnhubWebSocket) Subscribe(symbol string) {
	select {
	case f.subCh <- symbol:
	default:
		slog.Warn("Finnhub WS subscribe channel full", "symbol", symbol)
	}
}

func (f *FinnhubWebSocket) Unsubscribe(symbol string) {
	select {
	case f.unsubCh <- symbol:
	default:
	}
}

func (f *FinnhubWebSocket) runLoop(ctx context.Context) {
	for {
		err := f.connect(ctx)
		if err != nil {
			slog.Warn("Finnhub WS connect failed", "error", err, "backoff", f.backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(f.backoff):
			}
			f.backoff = min(f.backoff*2, 60*time.Second)
			continue
		}
		f.backoff = 1 * time.Second
		f.readLoop(ctx)
		if ctx.Err() != nil {
			return
		}
		slog.Info("Finnhub WS disconnected, reconnecting...")
	}
}

func (f *FinnhubWebSocket) connect(ctx context.Context) error {
	url := fmt.Sprintf("%s?token=%s", finnhubWSURL, f.apiKey)
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		return fmt.Errorf("dial Finnhub WS: %w", err)
	}

	f.mu.Lock()
	f.conn = conn
	existing := make([]string, 0, len(f.subscribed))
	for sym := range f.subscribed {
		existing = append(existing, sym)
	}
	f.mu.Unlock()

	for _, sym := range existing {
		f.sendSubscribe(conn, sym)
	}

	f.drainChannels(conn)
	slog.Info("Finnhub WS connected")
	return nil
}

func (f *FinnhubWebSocket) readLoop(ctx context.Context) {
	msgCh := make(chan []byte, 64)
	errCh := make(chan error, 1)

	f.mu.Lock()
	conn := f.conn
	f.mu.Unlock()

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			select {
			case msgCh <- msg:
			default:
				slog.Warn("Finnhub WS message buffer full, dropping tick")
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			f.mu.Lock()
			conn.Close()
			f.conn = nil
			f.mu.Unlock()
			return

		case err := <-errCh:
			slog.Warn("Finnhub WS read error", "error", err)
			f.mu.Lock()
			f.conn = nil
			f.mu.Unlock()
			return

		case msg := <-msgCh:
			f.handleMessage(msg)

		case sym := <-f.subCh:
			f.mu.Lock()
			f.subscribed[sym] = true
			f.mu.Unlock()
			f.sendSubscribe(conn, sym)

		case sym := <-f.unsubCh:
			f.mu.Lock()
			delete(f.subscribed, sym)
			delete(f.prevClose, sym)
			f.mu.Unlock()
			f.sendUnsubscribe(conn, sym)
		}
	}
}

func (f *FinnhubWebSocket) drainChannels(conn *websocket.Conn) {
	for {
		select {
		case sym := <-f.subCh:
			f.mu.Lock()
			f.subscribed[sym] = true
			f.mu.Unlock()
			f.sendSubscribe(conn, sym)
		case sym := <-f.unsubCh:
			f.mu.Lock()
			delete(f.subscribed, sym)
			f.mu.Unlock()
			f.sendUnsubscribe(conn, sym)
		default:
			return
		}
	}
}

func (f *FinnhubWebSocket) sendSubscribe(conn *websocket.Conn, symbol string) {
	msg := map[string]string{"type": "subscribe", "symbol": symbol}
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		slog.Warn("Finnhub WS subscribe failed", "symbol", symbol, "error", err)
	}
}

func (f *FinnhubWebSocket) sendUnsubscribe(conn *websocket.Conn, symbol string) {
	msg := map[string]string{"type": "unsubscribe", "symbol": symbol}
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		slog.Warn("Finnhub WS unsubscribe failed", "symbol", symbol, "error", err)
	}
}

// SetPreviousClose stores the previous close price for change calculation.
func (f *FinnhubWebSocket) SetPreviousClose(symbol string, price float64) {
	f.mu.Lock()
	f.prevClose[symbol] = price
	f.mu.Unlock()
}

func (f *FinnhubWebSocket) handleMessage(msg []byte) {
	var payload struct {
		Type string `json:"type"`
		Data []struct {
			Price     float64 `json:"p"`
			Symbol    string  `json:"s"`
			Timestamp int64   `json:"t"` // milliseconds
			Volume    float64 `json:"v"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg, &payload); err != nil {
		return
	}

	if payload.Type != "trade" || len(payload.Data) == 0 {
		return
	}

	// Finnhub sends multiple trades per message; use the latest per symbol
	latest := make(map[string]struct {
		price  float64
		volume float64
	})
	for _, trade := range payload.Data {
		latest[trade.Symbol] = struct {
			price  float64
			volume float64
		}{trade.Price, trade.Volume}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	for sym, trade := range latest {
		change := 0.0
		changePct := 0.0
		if pc, ok := f.prevClose[sym]; ok && pc != 0 {
			change = trade.price - pc
			changePct = (change / pc) * 100
		}

		f.hub.BroadcastPrice(sym, model.StockQuote{
			Symbol:        sym,
			Price:         trade.price,
			Change:        change,
			ChangePercent: changePct,
			Volume:        int64(trade.volume),
			Currency:      "USD",
		})
	}
}
