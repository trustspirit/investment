package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shinyoung/investment/internal/model"
)

type KISService struct {
	auth  *KISAuth
	yahoo *YahooService
}

func NewKISService(auth *KISAuth, yahoo *YahooService) *KISService {
	return &KISService{auth: auth, yahoo: yahoo}
}

// StripKRXSuffix removes .KS/.KQ to get the 6-digit KRX code.
func StripKRXSuffix(symbol string) string {
	s := strings.TrimSuffix(symbol, ".KS")
	s = strings.TrimSuffix(s, ".KQ")
	return s
}

// doKISRequest performs an authenticated GET to KIS with 401-retry.
func (k *KISService) doKISRequest(ctx context.Context, path string, params map[string]string, trID string) ([]byte, error) {
	body, statusCode, err := k.doKISRequestOnce(ctx, path, params, trID)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized {
		slog.Info("KIS 401, force-renewing token and retrying")
		if renewErr := k.auth.ForceRenewToken(ctx); renewErr != nil {
			return nil, fmt.Errorf("renew KIS token: %w", renewErr)
		}
		body, _, err = k.doKISRequestOnce(ctx, path, params, trID)
		return body, err
	}
	return body, nil
}

func (k *KISService) doKISRequestOnce(ctx context.Context, path string, params map[string]string, trID string) ([]byte, int, error) {
	tok, err := k.auth.GetToken(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("get KIS token: %w", err)
	}

	u, err := url.Parse(kisBaseURL + path)
	if err != nil {
		return nil, 0, fmt.Errorf("build KIS URL: %w", err)
	}
	q := u.Query()
	for key, val := range params {
		q.Set(key, val)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("appkey", k.auth.appKey)
	req.Header.Set("appsecret", k.auth.appSecret)
	req.Header.Set("tr_id", trID)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("KIS request to %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	return body, resp.StatusCode, err
}

// GetQuote fetches the current price for a Korean stock (e.g. "005930.KS").
// Change is calculated as price - today's opening price.
func (k *KISService) GetQuote(ctx context.Context, symbol string) (model.StockQuote, error) {
	code := StripKRXSuffix(symbol)

	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-price", map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
	}, "FHKST01010100")
	if err != nil {
		return model.StockQuote{}, fmt.Errorf("KIS GetQuote %s: %w", symbol, err)
	}

	var resp struct {
		Output struct {
			StckPrpr string `json:"stck_prpr"` // 현재가
			StckOprc string `json:"stck_oprc"` // 오늘 시가
			AcmlVol  string `json:"acml_vol"`  // 누적거래량
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return model.StockQuote{}, fmt.Errorf("parse KIS quote for %s: %w", symbol, err)
	}

	price := parseKISFloat(resp.Output.StckPrpr)
	open := parseKISFloat(resp.Output.StckOprc)
	change := price - open
	changePct := 0.0
	if open != 0 {
		changePct = (change / open) * 100
	}

	return model.StockQuote{
		Symbol:        symbol,
		Price:         price,
		Change:        change,
		ChangePercent: changePct,
		Volume:        parseKISInt64(resp.Output.AcmlVol),
		Currency:      "KRW",
	}, nil
}

// GetHistoricalData returns OHLCV data for the given range.
func (k *KISService) GetHistoricalData(ctx context.Context, symbol string, chartRange string) ([]model.HistoricalDataPoint, error) {
	switch chartRange {
	case "1d", "5d", "pre":
		return k.getIntradayData(ctx, symbol)
	default:
		return k.getDailyData(ctx, symbol, chartRange)
	}
}

func (k *KISService) getIntradayData(ctx context.Context, symbol string) ([]model.HistoricalDataPoint, error) {
	code := StripKRXSuffix(symbol)
	inputHour := time.Now().Format("150405")

	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice", map[string]string{
		"FID_ETC_CLS_CODE":       "0",
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
		"FID_INPUT_HOUR_1":       inputHour,
		"FID_PW_DATA_INCU_YN":    "Y",
	}, "FHKST03010200")
	if err != nil {
		return nil, fmt.Errorf("KIS intraday %s: %w", symbol, err)
	}

	var resp struct {
		Output2 []struct {
			Hour     string `json:"stck_cntg_hour"`
			StckOprc string `json:"stck_oprc"`
			StckHgpr string `json:"stck_hgpr"`
			StckLwpr string `json:"stck_lwpr"`
			StckPrpr string `json:"stck_prpr"`
			CntgVol  string `json:"cntg_vol"`
		} `json:"output2"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse KIS intraday: %w", err)
	}

	today := time.Now().Format("20060102")
	points := make([]model.HistoricalDataPoint, 0, len(resp.Output2))
	for _, item := range resp.Output2 {
		t, err := time.ParseInLocation("20060102150405", today+item.Hour, time.Local)
		if err != nil {
			continue
		}
		points = append(points, model.HistoricalDataPoint{
			Timestamp: t,
			Open:      parseKISFloat(item.StckOprc),
			High:      parseKISFloat(item.StckHgpr),
			Low:       parseKISFloat(item.StckLwpr),
			Close:     parseKISFloat(item.StckPrpr),
			Volume:    parseKISInt64(item.CntgVol),
		})
	}
	return points, nil
}

type kisDateRange struct {
	periodDivCode string
	count         int
}

var kisDailyRangeMap = map[string]kisDateRange{
	"1mo": {"D", 30},
	"6mo": {"D", 180},
	"1y":  {"D", 365},
	"5y":  {"W", 260},
	"max": {"M", 120},
}

func (k *KISService) getDailyData(ctx context.Context, symbol string, chartRange string) ([]model.HistoricalDataPoint, error) {
	rc, ok := kisDailyRangeMap[chartRange]
	if !ok {
		return nil, fmt.Errorf("unsupported range: %s", chartRange)
	}

	code := StripKRXSuffix(symbol)
	endDate := time.Now().Format("20060102")
	var startDate string
	switch rc.periodDivCode {
	case "W":
		startDate = time.Now().AddDate(0, 0, -(rc.count*7*2)).Format("20060102")
	case "M":
		startDate = time.Now().AddDate(0, -(rc.count*2), 0).Format("20060102")
	default:
		startDate = time.Now().AddDate(0, 0, -(rc.count*2)).Format("20060102")
	}

	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-daily-price", map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
		"FID_PERIOD_DIV_CODE":    rc.periodDivCode,
		"FID_ORG_ADJ_PRC":        "0",
		"FID_INPUT_DATE_1":       startDate,
		"FID_INPUT_DATE_2":       endDate,
	}, "FHKST01010400")
	if err != nil {
		return nil, fmt.Errorf("KIS daily %s: %w", symbol, err)
	}

	var resp struct {
		Output2 []struct {
			StckBsopDate string `json:"stck_bsop_date"`
			StckOprc     string `json:"stck_oprc"`
			StckHgpr     string `json:"stck_hgpr"`
			StckLwpr     string `json:"stck_lwpr"`
			StckClpr     string `json:"stck_clpr"`
			AcmlVol      string `json:"acml_vol"`
		} `json:"output2"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse KIS daily: %w", err)
	}

	points := make([]model.HistoricalDataPoint, 0, len(resp.Output2))
	for _, item := range resp.Output2 {
		t, err := time.ParseInLocation("20060102", item.StckBsopDate, time.Local)
		if err != nil {
			continue
		}
		points = append(points, model.HistoricalDataPoint{
			Timestamp: t,
			Open:      parseKISFloat(item.StckOprc),
			High:      parseKISFloat(item.StckHgpr),
			Low:       parseKISFloat(item.StckLwpr),
			Close:     parseKISFloat(item.StckClpr),
			Volume:    parseKISInt64(item.AcmlVol),
		})
	}
	// KIS returns newest-first; reverse to oldest-first
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	if len(points) > rc.count {
		points = points[len(points)-rc.count:]
	}
	return points, nil
}

// GetCompanyInfo returns basic company information for a Korean stock.
func (k *KISService) GetCompanyInfo(ctx context.Context, symbol string) (model.CompanyInfo, error) {
	code := StripKRXSuffix(symbol)

	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/search-stock-info", map[string]string{
		"PRDT_TYPE_CD": "300",
		"PDNO":         code,
	}, "CTPF1002R")
	if err != nil {
		return model.CompanyInfo{}, fmt.Errorf("KIS GetCompanyInfo %s: %w", symbol, err)
	}

	var resp struct {
		Output struct {
			PdnoName     string `json:"prdt_abrv_name"`
			EngName      string `json:"prdt_eng_name"`
			IdstClsfName string `json:"std_idst_clsf_cd_name"`
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return model.CompanyInfo{}, fmt.Errorf("parse KIS company info: %w", err)
	}

	name := resp.Output.PdnoName
	if name == "" {
		name = resp.Output.EngName
	}

	return model.CompanyInfo{
		Symbol:   symbol,
		Name:     name,
		Sector:   resp.Output.IdstClsfName,
		Industry: resp.Output.IdstClsfName,
		Currency: "KRW",
	}, nil
}

// GetRecommendation delegates to Yahoo (KIS does not provide analyst consensus).
func (k *KISService) GetRecommendation(ctx context.Context, symbol string) (model.RecommendationData, error) {
	return k.yahoo.GetRecommendation(ctx, symbol)
}

// GetNews delegates to Yahoo (KIS has no news API).
func (k *KISService) GetNews(ctx context.Context, symbol string) ([]model.NewsArticle, error) {
	return k.yahoo.GetNews(ctx, symbol)
}

func parseKISFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", "")
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}

func parseKISInt64(s string) int64 {
	s = strings.ReplaceAll(s, ",", "")
	i, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return i
}
