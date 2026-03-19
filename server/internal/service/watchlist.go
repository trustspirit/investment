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
	items []model.WatchlistItem
}

func NewWatchlistService() *WatchlistService {
	ws := &WatchlistService{}
	if err := ws.load(); err != nil {
		slog.Warn("failed to load watchlist", "error", err)
	}
	return ws
}

func (ws *WatchlistService) Add(symbol, name string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for _, item := range ws.items {
		if item.Symbol == symbol {
			return nil
		}
	}

	ws.items = append(ws.items, model.WatchlistItem{
		Symbol:  symbol,
		Name:    name,
		AddedAt: time.Now().UTC(),
	})

	return ws.persist()
}

func (ws *WatchlistService) Remove(symbol string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	for i, item := range ws.items {
		if item.Symbol == symbol {
			ws.items = append(ws.items[:i], ws.items[i+1:]...)
			break
		}
	}
	return ws.persist()
}

func (ws *WatchlistService) List() []model.WatchlistItem {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	out := make([]model.WatchlistItem, len(ws.items))
	copy(out, ws.items)
	return out
}

// Reorder sets the watchlist order to match the given symbol list.
func (ws *WatchlistService) Reorder(symbols []string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	lookup := make(map[string]model.WatchlistItem, len(ws.items))
	for _, item := range ws.items {
		lookup[item.Symbol] = item
	}

	reordered := make([]model.WatchlistItem, 0, len(symbols))
	for _, sym := range symbols {
		if item, ok := lookup[sym]; ok {
			reordered = append(reordered, item)
			delete(lookup, sym)
		}
	}
	// Append any items not in the new order (safety)
	for _, item := range lookup {
		reordered = append(reordered, item)
	}

	ws.items = reordered
	return ws.persist()
}

func (ws *WatchlistService) Symbols() []string {
	ws.mu.RLock()
	defer ws.mu.RUnlock()

	symbols := make([]string, len(ws.items))
	for i, item := range ws.items {
		symbols[i] = item.Symbol
	}
	return symbols
}

func (ws *WatchlistService) persist() error {
	dir := filepath.Dir(watchlistFilePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	data, err := json.MarshalIndent(ws.items, "", "  ")
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

	ws.items = list
	slog.Info("loaded watchlist", "count", len(ws.items))
	return nil
}
