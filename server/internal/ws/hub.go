package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/shinyoung/investment/internal/model"
)

// SymbolEvent is emitted when a client subscribes or the last client unsubscribes from a symbol.
type SymbolEvent struct {
	Symbol    string
	Subscribe bool // true = subscribe, false = unsubscribe (last subscriber left)
}

type Hub struct {
	clients      map[*Client]bool
	register     chan *Client
	unregister   chan *Client
	mu           sync.RWMutex
	subMu        sync.RWMutex
	subs         map[string]map[*Client]bool
	SymbolEvents chan SymbolEvent // buffered cap 64
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		subs:         make(map[string]map[*Client]bool),
		SymbolEvents: make(chan SymbolEvent, 64),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			slog.Info("websocket client connected", "clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

			var emptied []string
			h.subMu.Lock()
			for symbol, clients := range h.subs {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.subs, symbol)
					emptied = append(emptied, symbol)
				}
			}
			h.subMu.Unlock()

			for _, sym := range emptied {
				select {
				case h.SymbolEvents <- SymbolEvent{Symbol: sym, Subscribe: false}:
				default:
				}
			}
			slog.Info("websocket client disconnected", "clients", len(h.clients))
		}
	}
}

func (h *Hub) Register() chan<- *Client   { return h.register }
func (h *Hub) Unregister() chan<- *Client { return h.unregister }

func (h *Hub) Subscribe(client *Client, symbol string) {
	h.subMu.Lock()
	if h.subs[symbol] == nil {
		h.subs[symbol] = make(map[*Client]bool)
	}
	h.subs[symbol][client] = true
	client.subscriptions[symbol] = true
	h.subMu.Unlock() // release BEFORE emitting — avoids deadlock with BroadcastPrice

	slog.Info("client subscribed", "symbol", symbol)
	select {
	case h.SymbolEvents <- SymbolEvent{Symbol: symbol, Subscribe: true}:
	default:
	}
}

func (h *Hub) Unsubscribe(client *Client, symbol string) {
	h.subMu.Lock()
	lastSubscriber := false
	if clients, ok := h.subs[symbol]; ok {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.subs, symbol)
			lastSubscriber = true
		}
	}
	delete(client.subscriptions, symbol)
	h.subMu.Unlock() // release BEFORE emitting

	if lastSubscriber {
		select {
		case h.SymbolEvents <- SymbolEvent{Symbol: symbol, Subscribe: false}:
		default:
		}
	}
}

func (h *Hub) BroadcastPrice(symbol string, quote model.StockQuote) {
	h.subMu.RLock()
	clients := h.subs[symbol]
	h.subMu.RUnlock()

	if len(clients) == 0 {
		return
	}

	msg := model.WSPriceUpdateMessage{
		Type:   model.WSMessageTypePriceUpdate,
		Symbol: symbol,
		Quote:  quote,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("failed to marshal price update", "error", err)
		return
	}

	for client := range clients {
		select {
		case client.send <- data:
		default:
			slog.Warn("client send buffer full, dropping message")
		}
	}
}

func (h *Hub) SubscribedSymbols() []string {
	h.subMu.RLock()
	defer h.subMu.RUnlock()

	symbols := make([]string, 0, len(h.subs))
	for s := range h.subs {
		symbols = append(symbols, s)
	}
	return symbols
}
