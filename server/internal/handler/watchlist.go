package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/shinyoung/investment/internal/service"
)

type WatchlistHandler struct {
	watchlist *service.WatchlistService
}

func NewWatchlistHandler(watchlist *service.WatchlistService) *WatchlistHandler {
	return &WatchlistHandler{watchlist: watchlist}
}

func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.watchlist.List())
}

func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Symbol = strings.TrimSpace(req.Symbol)
	req.Name = strings.TrimSpace(req.Name)
	if req.Symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	if err := h.watchlist.Add(req.Symbol, req.Name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	if err := h.watchlist.Remove(symbol); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *WatchlistHandler) Reorder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Symbols []string `json:"symbols"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.watchlist.Reorder(req.Symbols); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
