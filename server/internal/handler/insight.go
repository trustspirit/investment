package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/shinyoung/investment/internal/service"
)

type InsightHandler struct {
	scheduler *service.InsightScheduler
}

func NewInsightHandler(scheduler *service.InsightScheduler) *InsightHandler {
	return &InsightHandler{scheduler: scheduler}
}

func (h *InsightHandler) Get(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	insight, ok := h.scheduler.GetCached(symbol)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	writeJSON(w, http.StatusOK, insight)
}

func (h *InsightHandler) Generate(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	insight, err := h.scheduler.GenerateForSymbol(r.Context(), symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, insight)
}

func (h *InsightHandler) GenerateStrategy(w http.ResponseWriter, r *http.Request) {
	symbol := strings.TrimSpace(chi.URLParam(r, "symbol"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "symbol is required")
		return
	}

	strategy, err := h.scheduler.GenerateStrategyForSymbol(r.Context(), symbol)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, strategy)
}
