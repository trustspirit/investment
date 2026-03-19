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
	ws    *KISWebSocket
}

func NewKISService(auth *KISAuth, yahoo *YahooService) *KISService {
	return &KISService{auth: auth, yahoo: yahoo}
}

// SetWebSocket sets the KIS WebSocket reference for overtime tick cache access.
func (k *KISService) SetWebSocket(ws *KISWebSocket) {
	k.ws = ws
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
		body, statusCode, err = k.doKISRequestOnce(ctx, path, params, trID)
		if err != nil {
			return nil, err
		}
	}
	if statusCode != http.StatusOK {
		slog.Warn("KIS non-200 response", "path", path, "status", statusCode, "body", string(body[:min(len(body), 300)]))
		return nil, fmt.Errorf("KIS %s returned HTTP %d", path, statusCode)
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
			StckSdpr string `json:"stck_sdpr"` // 전일종가(기준가)
			AcmlVol  string `json:"acml_vol"`  // 누적거래량
			HtsAvls  string `json:"hts_avls"`  // 시가총액(억)
		} `json:"output"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return model.StockQuote{}, fmt.Errorf("parse KIS quote for %s: %w", symbol, err)
	}

	// hts_avls is in 억원 (100M KRW), convert to raw KRW
	htsAvls := parseKISInt64(resp.Output.HtsAvls)
	marketCap := htsAvls * 100_000_000

	price := parseKISFloat(resp.Output.StckPrpr)
	prevClose := parseKISFloat(resp.Output.StckSdpr)

	// Calculate change from previous close — more reliable than prdy_vrss which can be 0
	var change, changePct float64
	if prevClose != 0 {
		change = price - prevClose
		changePct = (change / prevClose) * 100
	}

	quote := model.StockQuote{
		Symbol:        symbol,
		Price:         price,
		Change:        change,
		ChangePercent: changePct,
		Volume:        parseKISInt64(resp.Output.AcmlVol),
		MarketCap:     marketCap,
		Currency:      "KRW",
	}

	// Check overtime price — only outside regular trading hours (09:00–15:30 KST)
	now := time.Now()
	hhmmss := now.Format("150405")
	if hhmmss >= "153000" || hhmmss < "090000" {
		if otPrice, _, _, ok := k.getOvertimePrice(ctx, code); ok && otPrice != 0 && otPrice != quote.Price {
			quote.Price = otPrice
			if prevClose != 0 {
				quote.Change = otPrice - prevClose
				quote.ChangePercent = (quote.Change / prevClose) * 100
			}
		}
	}

	return quote, nil
}

// getOvertimePrice fetches the after-hours (시간외) price.
// Tries real-time overtime first, falls back to daily overtime history.
func (k *KISService) getOvertimePrice(ctx context.Context, code string) (price, change, changePct float64, ok bool) {
	// Try real-time overtime price first
	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-overtime-price", map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
	}, "FHPST02300000")
	if err == nil {
		var resp struct {
			Output struct {
				OvtmUntpPrpr     string `json:"ovtm_untp_prpr"`
				OvtmUntpPrdyVrss string `json:"ovtm_untp_prdy_vrss"`
				OvtmUntpPrdyCtrt string `json:"ovtm_untp_prdy_ctrt"`
				OvtmUntpVrssSign string `json:"ovtm_untp_prdy_vrss_sign"`
			} `json:"output"`
		}
		if json.Unmarshal(body, &resp) == nil {
			otPrice := parseKISFloat(resp.Output.OvtmUntpPrpr)
			if otPrice != 0 {
				otChange := parseKISFloat(resp.Output.OvtmUntpPrdyVrss)
				if resp.Output.OvtmUntpVrssSign == "4" || resp.Output.OvtmUntpVrssSign == "5" {
					otChange = -otChange
				}
				otChangePct := parseKISFloat(resp.Output.OvtmUntpPrdyCtrt)
				if otChange < 0 {
					otChangePct = -otChangePct
				}
				return otPrice, otChange, otChangePct, true
			}
		}
	}

	// Fallback: daily overtime history (retains data after overtime closes)
	// Fallback: daily overtime history
	body, err = k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-daily-overtimeprice", map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
	}, "FHPST02320000")
	if err != nil {
		return 0, 0, 0, false
	}

	var dailyResp struct {
		Output1 struct {
			OvtmUntpPrpr     string `json:"ovtm_untp_prpr"`
			OvtmUntpPrdyVrss string `json:"ovtm_untp_prdy_vrss"`
			OvtmUntpPrdyCtrt string `json:"ovtm_untp_prdy_ctrt"`
			OvtmUntpVrssSign string `json:"ovtm_untp_prdy_vrss_sign"`
		} `json:"output1"`
	}
	if err := json.Unmarshal(body, &dailyResp); err != nil {
		return 0, 0, 0, false
	}

	otPrice := parseKISFloat(dailyResp.Output1.OvtmUntpPrpr)
	if otPrice == 0 {
		return 0, 0, 0, false
	}

	otChange := parseKISFloat(dailyResp.Output1.OvtmUntpPrdyVrss)
	if dailyResp.Output1.OvtmUntpVrssSign == "4" || dailyResp.Output1.OvtmUntpVrssSign == "5" {
		otChange = -otChange
	}
	otChangePct := parseKISFloat(dailyResp.Output1.OvtmUntpPrdyCtrt)
	if otChange < 0 {
		otChangePct = -otChangePct
	}

	return otPrice, otChange, otChangePct, true
}

// GetHistoricalData returns OHLCV data for the given range.
func (k *KISService) GetHistoricalData(ctx context.Context, symbol string, chartRange string) ([]model.HistoricalDataPoint, error) {
	switch chartRange {
	case "1d", "pre":
		return k.getIntradayData(ctx, symbol)
	default:
		return k.getDailyData(ctx, symbol, chartRange)
	}
}

func (k *KISService) getIntradayData(ctx context.Context, symbol string) ([]model.HistoricalDataPoint, error) {
	code := StripKRXSuffix(symbol)
	today := time.Now().Format("20060102")
	marketOpen := "090000"
	marketClose := "153000"

	// Start pagination from market close (or now if market is still open)
	cursor := time.Now().Format("150405")
	if cursor > marketClose {
		cursor = marketClose
	}

	var allPoints []model.HistoricalDataPoint
	seen := make(map[string]bool) // deduplicate by hour

	// KIS returns ~30 records per call; paginate backwards to 09:00
	for i := 0; i < 15; i++ {
		body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-time-itemchartprice", map[string]string{
			"FID_ETC_CLS_CODE":       "0",
			"FID_COND_MRKT_DIV_CODE": "J",
			"FID_INPUT_ISCD":         code,
			"FID_INPUT_HOUR_1":       cursor,
			"FID_PW_DATA_INCU_YN":    "N",
		}, "FHKST03010200")
		if err != nil {
			if i > 0 {
				break
			}
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
			break
		}
		if len(resp.Output2) == 0 {
			break
		}

		reachedOpen := false
		for _, item := range resp.Output2 {
			if item.Hour < marketOpen {
				reachedOpen = true
				continue
			}
			if item.Hour > marketClose {
				continue
			}
			if seen[item.Hour] {
				continue
			}
			seen[item.Hour] = true
			t, err := time.ParseInLocation("20060102150405", today+item.Hour, time.Local)
			if err != nil {
				continue
			}
			allPoints = append(allPoints, model.HistoricalDataPoint{
				Timestamp: t,
				Open:      parseKISFloat(item.StckOprc),
				High:      parseKISFloat(item.StckHgpr),
				Low:       parseKISFloat(item.StckLwpr),
				Close:     parseKISFloat(item.StckPrpr),
				Volume:    parseKISInt64(item.CntgVol),
			})
		}

		if reachedOpen {
			break
		}

		oldest := resp.Output2[len(resp.Output2)-1].Hour
		if oldest >= cursor {
			break
		}
		cursor = oldest
	}

	// KIS returns newest-first; reverse to oldest-first
	for i, j := 0, len(allPoints)-1; i < j; i, j = i+1, j-1 {
		allPoints[i], allPoints[j] = allPoints[j], allPoints[i]
	}

	// Append after-hours data from multiple sources (best-effort)
	allPoints = append(allPoints, k.collectOvertimePoints(ctx, code, today, allPoints)...)

	return allPoints, nil
}

// getOvertimeConclusions fetches after-hours tick data from KIS.
func (k *KISService) getOvertimeConclusions(ctx context.Context, code, today string) []model.HistoricalDataPoint {
	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-time-overtimeconclusion", map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
		"FID_HOUR_CLS_CODE":      "1",
	}, "FHPST02310000")
	if err != nil {
		slog.Warn("KIS overtime conclusions failed", "code", code, "error", err)
		return nil
	}

	var resp struct {
		Output2 []struct {
			Hour    string `json:"stck_cntg_hour"` // 체결시각
			StckPrpr string `json:"stck_prpr"`      // 현재가
			CntgVol  string `json:"cntg_vol"`       // 체결 거래량
		} `json:"output2"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil
	}

	var points []model.HistoricalDataPoint
	seen := make(map[string]bool)
	for _, item := range resp.Output2 {
		if item.Hour == "" || seen[item.Hour] {
			continue
		}
		seen[item.Hour] = true
		t, err := time.ParseInLocation("20060102150405", today+item.Hour, time.Local)
		if err != nil {
			continue
		}
		price := parseKISFloat(item.StckPrpr)
		if price == 0 {
			continue
		}
		points = append(points, model.HistoricalDataPoint{
			Timestamp: t,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    parseKISInt64(item.CntgVol),
		})
	}

	// Reverse: KIS returns newest-first
	for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
		points[i], points[j] = points[j], points[i]
	}
	return points
}

// collectOvertimePoints gathers after-hours data from all available sources.
func (k *KISService) collectOvertimePoints(ctx context.Context, code, today string, regularPoints []model.HistoricalDataPoint) []model.HistoricalDataPoint {
	// 1. Try WebSocket cached ticks (best: real-time granularity)
	if k.ws != nil {
		ticks := k.ws.GetOvertimeTicks(code)
		if len(ticks) > 0 {
			points := make([]model.HistoricalDataPoint, 0, len(ticks))
			for _, t := range ticks {
				points = append(points, model.HistoricalDataPoint{
					Timestamp: t.Time,
					Open:      t.Price,
					High:      t.Price,
					Low:       t.Price,
					Close:     t.Price,
					Volume:    t.Volume,
				})
			}
			return points
		}
	}

	// 2. Try REST overtime conclusion ticks (works during overtime session)
	otPoints := k.getOvertimeConclusions(ctx, code, today)
	if len(otPoints) > 0 {
		return otPoints
	}

	// 3. Fallback: daily overtime OHLCV (persists after session closes)
	if len(regularPoints) == 0 {
		return nil
	}
	lastRegularPrice := regularPoints[len(regularPoints)-1].Close

	body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-daily-overtimeprice", map[string]string{
		"FID_COND_MRKT_DIV_CODE": "J",
		"FID_INPUT_ISCD":         code,
	}, "FHPST02320000")
	if err != nil {
		return nil
	}

	var resp struct {
		Output1 struct {
			OvtmUntpPrpr string `json:"ovtm_untp_prpr"`
		} `json:"output1"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return nil
	}

	otPrice := parseKISFloat(resp.Output1.OvtmUntpPrpr)
	if otPrice == 0 || otPrice == lastRegularPrice {
		return nil
	}

	otTime, _ := time.ParseInLocation("20060102150405", today+"154000", time.Local)
	return []model.HistoricalDataPoint{{
		Timestamp: otTime,
		Open:      otPrice,
		High:      otPrice,
		Low:       otPrice,
		Close:     otPrice,
		Volume:    0,
	}}
}

type kisDateRange struct {
	periodDivCode string
	count         int
}

var kisDailyRangeMap = map[string]kisDateRange{
	"5d":  {"D", 5},
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
	targetStart := time.Now()
	switch rc.periodDivCode {
	case "W":
		targetStart = time.Now().AddDate(0, 0, -(rc.count * 7))
	case "M":
		targetStart = time.Now().AddDate(0, -rc.count, 0)
	default:
		targetStart = time.Now().AddDate(0, 0, -(rc.count * 2))
	}
	startDate := targetStart.Format("20060102")

	// KIS returns max 100 records per call; paginate backwards
	var allPoints []model.HistoricalDataPoint
	curEndDate := time.Now().Format("20060102")

	for page := 0; page < 10; page++ {
		body, err := k.doKISRequest(ctx, "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice", map[string]string{
			"FID_COND_MRKT_DIV_CODE": "J",
			"FID_INPUT_ISCD":         code,
			"FID_PERIOD_DIV_CODE":    rc.periodDivCode,
			"FID_ORG_ADJ_PRC":        "0",
			"FID_INPUT_DATE_1":       startDate,
			"FID_INPUT_DATE_2":       curEndDate,
		}, "FHKST03010100")
		if err != nil {
			if page > 0 {
				break
			}
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
			break
		}
		if len(resp.Output2) == 0 {
			break
		}

		for _, item := range resp.Output2 {
			t, err := time.ParseInLocation("20060102", item.StckBsopDate, time.Local)
			if err != nil {
				continue
			}
			allPoints = append(allPoints, model.HistoricalDataPoint{
				Timestamp: t,
				Open:      parseKISFloat(item.StckOprc),
				High:      parseKISFloat(item.StckHgpr),
				Low:       parseKISFloat(item.StckLwpr),
				Close:     parseKISFloat(item.StckClpr),
				Volume:    parseKISInt64(item.AcmlVol),
			})
		}

		// If fewer than 100 records, no more pages
		if len(resp.Output2) < 100 {
			break
		}

		// Move end date to day before oldest record for next page
		oldest := resp.Output2[len(resp.Output2)-1].StckBsopDate
		oldestTime, err := time.ParseInLocation("20060102", oldest, time.Local)
		if err != nil {
			break
		}
		curEndDate = oldestTime.AddDate(0, 0, -1).Format("20060102")
		if curEndDate < startDate {
			break
		}
	}

	// KIS returns newest-first; reverse to oldest-first
	points := allPoints
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
