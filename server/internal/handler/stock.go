package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/shinyoung/investment/internal/model"
	"github.com/shinyoung/investment/internal/service"
)

type StockHandler struct {
	stocks *service.StockService
	news   *service.NewsService
}

func NewStockHandler(stocks *service.StockService, news *service.NewsService) *StockHandler {
	return &StockHandler{stocks: stocks, news: news}
}

func (h *StockHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter q is required")
		return
	}

	results, err := h.stocks.SearchSymbol(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (h *StockHandler) GetQuote(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	quote, err := h.stocks.GetQuote(r.Context(), symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, quote)
}

func (h *StockHandler) GetChart(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	chartRange := strings.TrimSpace(r.URL.Query().Get("range"))
	if chartRange == "" {
		chartRange = "1d"
	}

	points, err := h.stocks.GetHistoricalData(r.Context(), symbol, chartRange)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "unsupported range") {
			status = http.StatusBadRequest
		}
		writeError(w, status, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, points)
}

func (h *StockHandler) GetInfo(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	info, err := h.stocks.GetCompanyInfo(r.Context(), symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, info)
}

func (h *StockHandler) GetRecommendation(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	rec, err := h.stocks.GetRecommendation(r.Context(), symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, rec)
}

func (h *StockHandler) GetNews(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	var sector, industry string
	info, err := h.stocks.GetCompanyInfo(r.Context(), symbol)
	if err != nil {
		slog.Warn("failed to get company info for news context", "symbol", symbol, "error", err)
	} else {
		sector = info.Sector
		industry = info.Industry
	}

	news, err := h.news.GetAllNews(r.Context(), symbol, sector, industry)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, news)
}

func (h *StockHandler) GetMarketIndicators(w http.ResponseWriter, r *http.Request) {
	indicators, err := h.stocks.GetMarketIndicators(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, indicators)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode response", "status", status, "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, model.ErrorResponse{Error: msg})
}
