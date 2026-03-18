package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/shinyoung/investment/internal/model"
	ws "github.com/shinyoung/investment/internal/ws"
)

const kisWSURL = "ws://ops.koreainvestment.com:21000"

// OvertimeTick stores a single after-hours trade for chart rendering.
type OvertimeTick struct {
	Code   string
	Time   time.Time
	Price  float64
	Volume int64
}

type KISWebSocket struct {
	auth       *KISAuth
	hub        *ws.Hub
	conn       *websocket.Conn
	subscribed map[string]bool // KRX codes (no suffix)
	mu         sync.Mutex      // guards conn + subscribed
	subCh      chan string      // full symbols e.g. "005930.KS" (buffered 32)
	unsubCh    chan string      // full symbols (buffered 32)
	backoff    time.Duration   // starts 1s, max 60s

	// In-memory cache of today's overtime ticks for chart data
	otMu    sync.RWMutex
	otTicks map[string][]OvertimeTick // code -> ticks (sorted by time)
	otDate  string                     // "20060102" — reset on new day
}

func NewKISWebSocket(auth *KISAuth, hub *ws.Hub) *KISWebSocket {
	return &KISWebSocket{
		auth:       auth,
		hub:        hub,
		subscribed: make(map[string]bool),
		subCh:      make(chan string, 32),
		unsubCh:    make(chan string, 32),
		backoff:    1 * time.Second,
		otTicks:    make(map[string][]OvertimeTick),
		otDate:     time.Now().Format("20060102"),
	}
}

// GetOvertimeTicks returns cached after-hours ticks for a KRX code (6-digit).
func (k *KISWebSocket) GetOvertimeTicks(code string) []OvertimeTick {
	k.otMu.RLock()
	defer k.otMu.RUnlock()
	return k.otTicks[code]
}

// Start begins the connection loop in a goroutine.
func (k *KISWebSocket) Start(ctx context.Context) {
	go k.runLoop(ctx)
}

// Subscribe enqueues a symbol for KIS WebSocket subscription.
func (k *KISWebSocket) Subscribe(symbol string) {
	select {
	case k.subCh <- symbol:
	default:
		slog.Warn("KIS WS subscribe channel full", "symbol", symbol)
	}
}

// Unsubscribe enqueues a symbol for KIS WebSocket unsubscription.
func (k *KISWebSocket) Unsubscribe(symbol string) {
	select {
	case k.unsubCh <- symbol:
	default:
	}
}

func (k *KISWebSocket) runLoop(ctx context.Context) {
	for {
		approvalKey, err := k.connect(ctx)
		if err != nil {
			slog.Warn("KIS WS connect failed", "error", err, "backoff", k.backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(k.backoff):
			}
			k.backoff = min(k.backoff*2, 60*time.Second)
			continue
		}
		k.backoff = 1 * time.Second
		k.readLoop(ctx, approvalKey)
		if ctx.Err() != nil {
			return
		}
		slog.Info("KIS WS disconnected, reconnecting...")
	}
}

// connect establishes the WebSocket connection and returns the approval key for use in readLoop.
func (k *KISWebSocket) connect(ctx context.Context) (string, error) {
	approvalKey, err := k.auth.GetWSApprovalKey(ctx)
	if err != nil {
		return "", fmt.Errorf("get WS approval key: %w", err)
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, kisWSURL, nil)
	if err != nil {
		return "", fmt.Errorf("dial KIS WS: %w", err)
	}

	k.mu.Lock()
	k.conn = conn
	existing := make([]string, 0, len(k.subscribed))
	for code := range k.subscribed {
		existing = append(existing, code)
	}
	k.mu.Unlock()

	// Resubscribe all previously subscribed symbols
	for _, code := range existing {
		if err := k.sendSubscribe(conn, approvalKey, code); err != nil {
			slog.Warn("KIS WS resubscribe failed", "code", code, "error", err)
		}
	}

	k.drainChannels(conn, approvalKey)
	slog.Info("KIS WS connected")
	return approvalKey, nil
}

// readLoop uses a dedicated goroutine for blocking reads and a select for channel events.
// approvalKey is passed from connect() to avoid a redundant HTTP call.
func (k *KISWebSocket) readLoop(ctx context.Context, approvalKey string) {
	msgCh := make(chan []byte, 16)
	errCh := make(chan error, 1)

	k.mu.Lock()
	conn := k.conn
	k.mu.Unlock()

	// Blocking read goroutine
	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			select {
			case msgCh <- msg:
			default:
				slog.Warn("KIS WS message buffer full, dropping tick")
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			k.mu.Lock()
			conn.Close()
			k.conn = nil
			k.mu.Unlock()
			return

		case err := <-errCh:
			slog.Warn("KIS WS read error", "error", err)
			k.mu.Lock()
			k.conn = nil
			k.mu.Unlock()
			return

		case msg := <-msgCh:
			k.handleMessage(msg)

		case sym := <-k.subCh:
			code := StripKRXSuffix(sym)
			k.mu.Lock()
			k.subscribed[code] = true
			k.mu.Unlock()
			k.sendSubscribe(conn, approvalKey, code) //nolint:errcheck

		case sym := <-k.unsubCh:
			code := StripKRXSuffix(sym)
			k.mu.Lock()
			delete(k.subscribed, code)
			k.mu.Unlock()
			k.sendUnsubscribe(conn, approvalKey, code) //nolint:errcheck
		}
	}
}

func (k *KISWebSocket) drainChannels(conn *websocket.Conn, approvalKey string) {
	for {
		select {
		case sym := <-k.subCh:
			code := StripKRXSuffix(sym)
			k.mu.Lock()
			k.subscribed[code] = true
			k.mu.Unlock()
			k.sendSubscribe(conn, approvalKey, code) //nolint:errcheck
		case sym := <-k.unsubCh:
			code := StripKRXSuffix(sym)
			k.mu.Lock()
			delete(k.subscribed, code)
			k.mu.Unlock()
			k.sendUnsubscribe(conn, approvalKey, code) //nolint:errcheck
		default:
			return
		}
	}
}

func (k *KISWebSocket) sendSubscribe(conn *websocket.Conn, approvalKey, code string) error {
	// Subscribe to both regular session and after-hours
	if err := k.sendTRRequest(conn, approvalKey, code, "1", "H0STCNT0"); err != nil {
		return err
	}
	return k.sendTRRequest(conn, approvalKey, code, "1", "H0STOUP0")
}

func (k *KISWebSocket) sendUnsubscribe(conn *websocket.Conn, approvalKey, code string) error {
	k.sendTRRequest(conn, approvalKey, code, "2", "H0STOUP0") //nolint:errcheck
	return k.sendTRRequest(conn, approvalKey, code, "2", "H0STCNT0")
}

func (k *KISWebSocket) sendTRRequest(conn *websocket.Conn, approvalKey, code, trType, trID string) error {
	msg := map[string]any{
		"header": map[string]string{
			"approval_key": approvalKey,
			"custtype":     "P",
			"tr_type":      trType,
			"content-type": "utf-8",
		},
		"body": map[string]any{
			"input": map[string]string{
				"tr_id":  trID,
				"tr_key": code,
			},
		},
	}
	data, _ := json.Marshal(msg)
	return conn.WriteMessage(websocket.TextMessage, data)
}

func (k *KISWebSocket) handleMessage(msg []byte) {
	// JSON frames are status/control responses — ignore
	if len(msg) > 0 && msg[0] == '{' {
		return
	}

	// Format: RECVTYPE|TRID|DATACNT|DATA
	parts := strings.SplitN(string(msg), "|", 4)
	if len(parts) < 4 {
		return
	}

	trID := parts[1]
	fields := strings.Split(parts[3], "^")

	var code string
	var price, change, changePct float64
	var volume int64

	switch trID {
	case "H0STCNT0": // 정규장 체결
		if len(fields) < 14 {
			return
		}
		code = fields[0]
		price = parseKISFloat(fields[2])
		changeAbs := parseKISFloat(fields[3])
		sign := fields[4]
		volume = parseKISInt64(fields[13])

		change = changeAbs
		if sign == "4" || sign == "5" {
			change = -changeAbs
		}

	case "H0STOUP0": // 시간외 체결
		if len(fields) < 14 {
			return
		}
		code = fields[0]
		price = parseKISFloat(fields[2])
		sign := fields[3]                     // 전일대비부호 (H0STOUP0: index 3)
		changeAbs := parseKISFloat(fields[4]) // 전일대비 (H0STOUP0: index 4)
		volume = parseKISInt64(fields[13])    // 누적거래량

		// Cache overtime tick for chart data
		k.cacheOvertimeTick(code, price, parseKISInt64(fields[12]))

		change = changeAbs
		if sign == "4" || sign == "5" {
			change = -changeAbs
		}

	default:
		return
	}

	prevClose := price - change
	if prevClose != 0 {
		changePct = (change / prevClose) * 100
	}

	// Broadcast for both .KS and .KQ — hub delivers only to subscribed clients
	for _, suffix := range []string{".KS", ".KQ"} {
		sym := code + suffix
		k.hub.BroadcastPrice(sym, model.StockQuote{
			Symbol:        sym,
			Price:         price,
			Change:        change,
			ChangePercent: changePct,
			Volume:        volume,
			Currency:      "KRW",
		})
	}
}

func (k *KISWebSocket) cacheOvertimeTick(code string, price float64, volume int64) {
	k.otMu.Lock()
	defer k.otMu.Unlock()

	// Reset cache on new day
	today := time.Now().Format("20060102")
	if today != k.otDate {
		k.otTicks = make(map[string][]OvertimeTick)
		k.otDate = today
	}

	k.otTicks[code] = append(k.otTicks[code], OvertimeTick{
		Code:   code,
		Time:   time.Now(),
		Price:  price,
		Volume: volume,
	})
}
