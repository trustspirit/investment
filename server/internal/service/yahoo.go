package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/shinyoung/investment/internal/model"
)

const (
	yahooChartURL  = "https://query1.finance.yahoo.com/v8/finance/chart"
	yahooQuoteURL  = "https://query1.finance.yahoo.com/v10/finance/quoteSummary"
	yahooSearchURL = "https://query1.finance.yahoo.com/v1/finance/search"
	yahooCookieURL = "https://fc.yahoo.com"
	yahooCrumbURL  = "https://query2.finance.yahoo.com/v1/test/getcrumb"
	yahooUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)"
)

type YahooService struct {
	client *http.Client
	mu     sync.RWMutex
	crumb  string
	cookie string
}

func NewYahooService() *YahooService {
	jar, _ := cookiejar.New(nil)
	return &YahooService{
		client: &http.Client{Timeout: 15 * time.Second, Jar: jar},
	}
}

func (s *YahooService) fetchCrumb(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cookieReq, err := http.NewRequestWithContext(ctx, http.MethodGet, yahooCookieURL, nil)
	if err != nil {
		return fmt.Errorf("create cookie request: %w", err)
	}
	cookieReq.Header.Set("User-Agent", yahooUserAgent)

	cookieResp, err := s.client.Do(cookieReq)
	if err != nil {
		return fmt.Errorf("fetch yahoo cookie: %w", err)
	}
	defer cookieResp.Body.Close()
	io.Copy(io.Discard, cookieResp.Body)

	var a3Cookie string
	for _, c := range s.client.Jar.Cookies(cookieReq.URL) {
		if c.Name == "A3" {
			a3Cookie = c.Name + "=" + c.Value
			break
		}
	}
	if a3Cookie == "" {
		return fmt.Errorf("no A3 cookie returned from yahoo")
	}

	crumbReq, err := http.NewRequestWithContext(ctx, http.MethodGet, yahooCrumbURL, nil)
	if err != nil {
		return fmt.Errorf("create crumb request: %w", err)
	}
	crumbReq.Header.Set("User-Agent", yahooUserAgent)

	crumbResp, err := s.client.Do(crumbReq)
	if err != nil {
		return fmt.Errorf("fetch yahoo crumb: %w", err)
	}
	defer crumbResp.Body.Close()

	crumbBytes, err := io.ReadAll(crumbResp.Body)
	if err != nil {
		return fmt.Errorf("read crumb response: %w", err)
	}

	crumb := strings.TrimSpace(string(crumbBytes))
	if crumb == "" || strings.Contains(crumb, "Unauthorized") {
		return fmt.Errorf("failed to get valid crumb: %s", crumb)
	}

	s.crumb = crumb
	s.cookie = a3Cookie
	slog.Info("fetched yahoo crumb", "crumb", crumb[:min(len(crumb), 4)]+"...")
	return nil
}

func (s *YahooService) getCrumb() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.crumb, s.cookie
}

func (s *YahooService) ensureCrumb(ctx context.Context) error {
	crumb, _ := s.getCrumb()
	if crumb != "" {
		return nil
	}
	return s.fetchCrumb(ctx)
}

type rangeConfig struct {
	Range    string
	Interval string
}

var rangeMap = map[string]rangeConfig{
	"pre": {Range: "1d", Interval: "1m"},
	"1d":  {Range: "1d", Interval: "5m"},
	"5d":  {Range: "5d", Interval: "15m"},
	"1mo": {Range: "1mo", Interval: "30m"},
	"6mo": {Range: "6mo", Interval: "1d"},
	"1y":  {Range: "1y", Interval: "1d"},
	"5y":  {Range: "5y", Interval: "1wk"},
	"max": {Range: "max", Interval: "1mo"},
}

func (s *YahooService) doRequest(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", yahooUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("yahoo API returned status %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	return body, nil
}

func (s *YahooService) doAuthenticatedRequest(ctx context.Context, rawURL string) ([]byte, error) {
	if err := s.ensureCrumb(ctx); err != nil {
		return nil, fmt.Errorf("ensure crumb: %w", err)
	}

	crumb, _ := s.getCrumb()
	separator := "?"
	if strings.Contains(rawURL, "?") {
		separator = "&"
	}
	authedURL := rawURL + separator + "crumb=" + url.QueryEscape(crumb)

	body, err := s.doRequest(ctx, authedURL)
	if err != nil && strings.Contains(err.Error(), "401") {
		slog.Info("crumb expired, refreshing")
		if fetchErr := s.fetchCrumb(ctx); fetchErr != nil {
			return nil, fmt.Errorf("refresh crumb: %w", fetchErr)
		}
		crumb, _ = s.getCrumb()
		authedURL = rawURL + separator + "crumb=" + url.QueryEscape(crumb)
		return s.doRequest(ctx, authedURL)
	}

	return body, err
}

func (s *YahooService) GetQuote(ctx context.Context, symbol string) (model.StockQuote, error) {
	u := fmt.Sprintf("%s/%s?modules=price,summaryDetail", yahooQuoteURL, url.PathEscape(symbol))

	body, err := s.doAuthenticatedRequest(ctx, u)
	if err != nil {
		return model.StockQuote{}, fmt.Errorf("get quote for %s: %w", symbol, err)
	}

	var resp struct {
		QuoteSummary struct {
			Result []struct {
				Price struct {
					ShortName                  string                `json:"shortName"`
					Currency                   string                `json:"currency"`
					RegularMarketPrice         struct{ Raw float64 } `json:"regularMarketPrice"`
					RegularMarketChange        struct{ Raw float64 } `json:"regularMarketChange"`
					RegularMarketChangePercent struct{ Raw float64 } `json:"regularMarketChangePercent"`
					RegularMarketVolume        struct{ Raw int64 }   `json:"regularMarketVolume"`
					MarketCap                  struct{ Raw int64 }   `json:"marketCap"`
					PreMarketPrice             struct{ Raw float64 } `json:"preMarketPrice"`
					PostMarketPrice            struct{ Raw float64 } `json:"postMarketPrice"`
				} `json:"price"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return model.StockQuote{}, fmt.Errorf("parse quote response for %s: %w", symbol, err)
	}

	if resp.QuoteSummary.Error != nil {
		return model.StockQuote{}, fmt.Errorf("yahoo error for %s: %s", symbol, resp.QuoteSummary.Error.Description)
	}

	if len(resp.QuoteSummary.Result) == 0 {
		return model.StockQuote{}, fmt.Errorf("no quote data for %s", symbol)
	}

	p := resp.QuoteSummary.Result[0].Price
	currency := p.Currency
	if currency == "" {
		currency = "USD"
	}
	quote := model.StockQuote{
		Symbol:        symbol,
		Name:          p.ShortName,
		Price:         p.RegularMarketPrice.Raw,
		Change:        p.RegularMarketChange.Raw,
		ChangePercent: p.RegularMarketChangePercent.Raw,
		Volume:        p.RegularMarketVolume.Raw,
		MarketCap:     p.MarketCap.Raw,
		Currency:      currency,
	}

	if p.PreMarketPrice.Raw != 0 {
		pre := p.PreMarketPrice.Raw
		quote.PreMarket = &pre
	}
	if p.PostMarketPrice.Raw != 0 {
		post := p.PostMarketPrice.Raw
		quote.PostMarket = &post
	}

	return quote, nil
}

func (s *YahooService) GetHistoricalData(ctx context.Context, symbol string, chartRange string) ([]model.HistoricalDataPoint, error) {
	rc, ok := rangeMap[chartRange]
	if !ok {
		return nil, fmt.Errorf("unsupported range: %s", chartRange)
	}

	u := fmt.Sprintf("%s/%s?range=%s&interval=%s&includePrePost=true", yahooChartURL, url.PathEscape(symbol), rc.Range, rc.Interval)

	body, err := s.doRequest(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("get chart for %s: %w", symbol, err)
	}

	var resp struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Open   []float64 `json:"open"`
						High   []float64 `json:"high"`
						Low    []float64 `json:"low"`
						Close  []float64 `json:"close"`
						Volume []int64   `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
				Meta struct {
					RegularMarketTime int64  `json:"regularMarketTime"`
					Timezone          string `json:"timezone"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse chart response for %s: %w", symbol, err)
	}

	if resp.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo chart error for %s: %s", symbol, resp.Chart.Error.Description)
	}

	if len(resp.Chart.Result) == 0 || len(resp.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no chart data for %s", symbol)
	}

	r := resp.Chart.Result[0]
	q := r.Indicators.Quote[0]
	points := make([]model.HistoricalDataPoint, 0, len(r.Timestamp))

	for i, ts := range r.Timestamp {
		if i >= len(q.Close) || i >= len(q.Open) || i >= len(q.High) || i >= len(q.Low) {
			continue
		}

		t := time.Unix(ts, 0).UTC()

		if chartRange == "pre" {
			hour := t.Hour()
			if hour >= 13 && hour <= 23 {
				continue
			}
		}

		points = append(points, model.HistoricalDataPoint{
			Timestamp: t,
			Open:      q.Open[i],
			High:      q.High[i],
			Low:       q.Low[i],
			Close:     q.Close[i],
			Volume:    safeVolumeAt(q.Volume, i),
		})
	}

	return points, nil
}

func (s *YahooService) GetCompanyInfo(ctx context.Context, symbol string) (model.CompanyInfo, error) {
	modules := "assetProfile,defaultKeyStatistics,summaryDetail,price"
	u := fmt.Sprintf("%s/%s?modules=%s", yahooQuoteURL, url.PathEscape(symbol), modules)

	body, err := s.doAuthenticatedRequest(ctx, u)
	if err != nil {
		return model.CompanyInfo{}, fmt.Errorf("get company info for %s: %w", symbol, err)
	}

	var resp struct {
		QuoteSummary struct {
			Result []struct {
				AssetProfile struct {
					Sector              string `json:"sector"`
					Industry            string `json:"industry"`
					LongBusinessSummary string `json:"longBusinessSummary"`
					FullTimeEmployees   int64  `json:"fullTimeEmployees"`
					Website             string `json:"website"`
				} `json:"assetProfile"`
				DefaultKeyStatistics struct {
					TrailingEps struct{ Raw float64 } `json:"trailingEps"`
					PegRatio    struct{ Raw float64 } `json:"pegRatio"`
					Beta        struct{ Raw float64 } `json:"beta"`
				} `json:"defaultKeyStatistics"`
				SummaryDetail struct {
					TrailingPE       struct{ Raw float64 } `json:"trailingPE"`
					DividendYield    struct{ Raw float64 } `json:"dividendYield"`
					FiftyTwoWeekHigh struct{ Raw float64 } `json:"fiftyTwoWeekHigh"`
					FiftyTwoWeekLow  struct{ Raw float64 } `json:"fiftyTwoWeekLow"`
					MarketCap        struct{ Raw int64 }   `json:"marketCap"`
				} `json:"summaryDetail"`
				Price struct {
					ShortName string `json:"shortName"`
					Currency  string `json:"currency"`
				} `json:"price"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return model.CompanyInfo{}, fmt.Errorf("parse company info for %s: %w", symbol, err)
	}

	if resp.QuoteSummary.Error != nil {
		return model.CompanyInfo{}, fmt.Errorf("yahoo error for %s: %s", symbol, resp.QuoteSummary.Error.Description)
	}

	if len(resp.QuoteSummary.Result) == 0 {
		return model.CompanyInfo{}, fmt.Errorf("no company info for %s", symbol)
	}

	r := resp.QuoteSummary.Result[0]
	companyCurrency := r.Price.Currency
	if companyCurrency == "" {
		companyCurrency = "USD"
	}
	info := model.CompanyInfo{
		Symbol:      symbol,
		Name:        r.Price.ShortName,
		Sector:      r.AssetProfile.Sector,
		Industry:    r.AssetProfile.Industry,
		Description: r.AssetProfile.LongBusinessSummary,
		Employees:   r.AssetProfile.FullTimeEmployees,
		Website:     r.AssetProfile.Website,
		Currency:    companyCurrency,
	}

	if v := r.SummaryDetail.TrailingPE.Raw; v != 0 {
		info.PE = &v
	}
	if v := r.DefaultKeyStatistics.TrailingEps.Raw; v != 0 {
		info.EPS = &v
	}
	if v := r.SummaryDetail.DividendYield.Raw; v != 0 {
		info.DividendYield = &v
	}
	if v := r.SummaryDetail.FiftyTwoWeekHigh.Raw; v != 0 {
		info.FiftyTwoWeekHigh = &v
	}
	if v := r.SummaryDetail.FiftyTwoWeekLow.Raw; v != 0 {
		info.FiftyTwoWeekLow = &v
	}
	if v := r.DefaultKeyStatistics.Beta.Raw; v != 0 {
		info.Beta = &v
	}

	return info, nil
}

func (s *YahooService) SearchSymbol(ctx context.Context, query string) ([]model.SymbolSearchResult, error) {
	u := fmt.Sprintf("%s?q=%s&quotesCount=10&newsCount=0&listsCount=0", yahooSearchURL, url.QueryEscape(query))

	body, err := s.doRequest(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("search symbols for %q: %w", query, err)
	}

	var resp struct {
		Quotes []struct {
			Symbol    string `json:"symbol"`
			ShortName string `json:"shortname"`
			Exchange  string `json:"exchange"`
			QuoteType string `json:"quoteType"`
		} `json:"quotes"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}

	results := make([]model.SymbolSearchResult, 0, len(resp.Quotes))
	for _, q := range resp.Quotes {
		if q.QuoteType == "EQUITY" || q.QuoteType == "ETF" {
			results = append(results, model.SymbolSearchResult{
				Symbol:   q.Symbol,
				Name:     q.ShortName,
				Exchange: q.Exchange,
				Type:     strings.ToLower(q.QuoteType),
			})
		}
	}

	return results, nil
}

func (s *YahooService) GetNews(ctx context.Context, symbol string) ([]model.NewsArticle, error) {
	u := fmt.Sprintf("%s?q=%s&newsCount=15&quotesCount=0&listsCount=0", yahooSearchURL, url.QueryEscape(symbol))

	body, err := s.doRequest(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("get news for %s: %w", symbol, err)
	}

	var resp struct {
		News []struct {
			Title               string `json:"title"`
			Link                string `json:"link"`
			Publisher           string `json:"publisher"`
			ProviderPublishTime int64  `json:"providerPublishTime"`
			Thumbnail           struct {
				Resolutions []struct {
					URL string `json:"url"`
				} `json:"resolutions"`
			} `json:"thumbnail"`
			RelatedTickers []string `json:"relatedTickers"`
		} `json:"news"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("parse news response for %s: %w", symbol, err)
	}

	articles := make([]model.NewsArticle, 0, len(resp.News))
	for _, n := range resp.News {
		thumbnail := ""
		if len(n.Thumbnail.Resolutions) > 0 {
			thumbnail = n.Thumbnail.Resolutions[0].URL
		}

		articles = append(articles, model.NewsArticle{
			Title:          n.Title,
			Link:           n.Link,
			Source:         n.Publisher,
			PublishedAt:    time.Unix(n.ProviderPublishTime, 0).UTC(),
			Thumbnail:      thumbnail,
			RelatedSymbols: n.RelatedTickers,
			Category:       model.NewsCategoryCompany,
		})
	}

	slog.Info("fetched news", "symbol", symbol, "count", len(articles))
	return articles, nil
}

func (s *YahooService) GetRecommendation(ctx context.Context, symbol string) (model.RecommendationData, error) {
	modules := "recommendationTrend,financialData,price"
	u := fmt.Sprintf("%s/%s?modules=%s", yahooQuoteURL, url.PathEscape(symbol), modules)

	body, err := s.doAuthenticatedRequest(ctx, u)
	if err != nil {
		return model.RecommendationData{}, fmt.Errorf("get recommendation for %s: %w", symbol, err)
	}

	var resp struct {
		QuoteSummary struct {
			Result []struct {
				RecommendationTrend struct {
					Trend []struct {
						Period     string `json:"period"`
						StrongBuy  int    `json:"strongBuy"`
						Buy        int    `json:"buy"`
						Hold       int    `json:"hold"`
						Sell       int    `json:"sell"`
						StrongSell int    `json:"strongSell"`
					} `json:"trend"`
				} `json:"recommendationTrend"`
				FinancialData struct {
					RecommendationKey       string                `json:"recommendationKey"`
					RecommendationMean      struct{ Raw float64 } `json:"recommendationMean"`
					NumberOfAnalystOpinions struct{ Raw int }     `json:"numberOfAnalystOpinions"`
					TargetMeanPrice         struct{ Raw float64 } `json:"targetMeanPrice"`
					TargetHighPrice         struct{ Raw float64 } `json:"targetHighPrice"`
					TargetLowPrice          struct{ Raw float64 } `json:"targetLowPrice"`
					CurrentPrice            struct{ Raw float64 } `json:"currentPrice"`
				} `json:"financialData"`
				Price struct {
					Currency string `json:"currency"`
				} `json:"price"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"quoteSummary"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return model.RecommendationData{}, fmt.Errorf("parse recommendation response for %s: %w", symbol, err)
	}

	if resp.QuoteSummary.Error != nil {
		return model.RecommendationData{}, fmt.Errorf("yahoo error for %s: %s", symbol, resp.QuoteSummary.Error.Description)
	}

	if len(resp.QuoteSummary.Result) == 0 {
		return model.RecommendationData{}, fmt.Errorf("no recommendation data for %s", symbol)
	}

	r := resp.QuoteSummary.Result[0]
	fd := r.FinancialData
	currency := r.Price.Currency
	if currency == "" {
		currency = "USD"
	}

	data := model.RecommendationData{
		Symbol:             symbol,
		RecommendationKey:  fd.RecommendationKey,
		RecommendationMean: fd.RecommendationMean.Raw,
		NumberOfAnalysts:   fd.NumberOfAnalystOpinions.Raw,
		CurrentPrice:       fd.CurrentPrice.Raw,
		Currency:           currency,
	}

	if v := fd.TargetMeanPrice.Raw; v != 0 {
		data.TargetMeanPrice = &v
	}
	if v := fd.TargetHighPrice.Raw; v != 0 {
		data.TargetHighPrice = &v
	}
	if v := fd.TargetLowPrice.Raw; v != 0 {
		data.TargetLowPrice = &v
	}

	trends := make([]model.RecommendationTrend, 0, len(r.RecommendationTrend.Trend))
	for _, t := range r.RecommendationTrend.Trend {
		trends = append(trends, model.RecommendationTrend{
			Period:     t.Period,
			StrongBuy:  t.StrongBuy,
			Buy:        t.Buy,
			Hold:       t.Hold,
			Sell:       t.Sell,
			StrongSell: t.StrongSell,
		})
	}
	data.Trend = trends

	slog.Info("fetched recommendation", "symbol", symbol, "key", data.RecommendationKey, "analysts", data.NumberOfAnalysts)
	return data, nil
}

func safeVolumeAt(volumes []int64, i int) int64 {
	if i < len(volumes) {
		return volumes[i]
	}
	return 0
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
