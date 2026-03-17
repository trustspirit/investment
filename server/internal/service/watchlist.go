package service

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shinyoung/investment/internal/model"
)

const watchlistFilePath = "data/watchlist.json"

type WatchlistService struct {
	mu    sync.RWMutex
	items map[string]model.WatchlistItem
}

func NewWatchlistService() *WatchlistService {
	ws := &WatchlistService{
		items: make(map[string]model.WatchlistItem),
	}
	if err := ws.load(); err != nil {
		slog.Warn("failed to load watchlist", "error", err)
	}
	return ws
}

func (ws *WatchlistService) Add(symbol, name string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if _, exists := ws.items[symbol]; exists {
		return nil
	}

	ws.items[symbol] = model.WatchlistItem{
		Symbol:  symbol,
		Name:    name,
		AddedAt: time.Now().UTC(),
	}

	return ws.persist()
}

func (ws *WatchlistService) Remove(symbol string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	delete(ws.items, symbol)
	return ws.persist()
}

func (ws *WatchlistService) List() []model.WatchlistItem {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	list := make([]model.WatchlistItem, 0, len(ws.items))
	for _, item := range ws.items {
		list = append(list, item)
	}
	return list
}

func (ws *WatchlistService) Symbols() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	symbols := make([]string, 0, len(ws.items))
	for s := range ws.items {
		symbols = append(symbols, s)
	}
	return symbols
}

func (ws *WatchlistService) persist() error {
	dir := filepath.Dir(watchlistFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	list := make([]model.WatchlistItem, 0, len(ws.items))
	for _, item := range ws.items {
		list = append(list, item)
	}

	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal watchlist: %w", err)
	}

	if err := os.WriteFile(watchlistFilePath, data, 0o644); err != nil {
		return fmt.Errorf("write watchlist file: %w", err)
	}

	return nil
}

func (ws *WatchlistService) load() error {
	data, err := os.ReadFile(watchlistFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read watchlist file: %w", err)
	}

	var list []model.WatchlistItem
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("unmarshal watchlist: %w", err)
	}

	for _, item := range list {
		ws.items[item.Symbol] = item
	}

	slog.Info("loaded watchlist", "count", len(ws.items))
	return nil
}
