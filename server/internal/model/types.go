package model

import "time"

type StockQuote struct {
	Symbol        string   `json:"symbol"`
	Name          string   `json:"name"`
	Price         float64  `json:"price"`
	Change        float64  `json:"change"`
	ChangePercent float64  `json:"changePercent"`
	Volume        int64    `json:"volume"`
	MarketCap     int64    `json:"marketCap"`
	Currency      string   `json:"currency"`
	PreMarket     *float64 `json:"preMarket,omitempty"`
	PostMarket    *float64 `json:"postMarket,omitempty"`
}

type HistoricalDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    int64     `json:"volume"`
}

type NewsArticle struct {
	Title          string    `json:"title"`
	Link           string    `json:"link"`
	Source         string    `json:"source"`
	PublishedAt    time.Time `json:"publishedAt"`
	Thumbnail      string    `json:"thumbnail"`
	RelatedSymbols []string  `json:"relatedSymbols"`
	Category       string    `json:"category"`
}

type CompanyInfo struct {
	Symbol           string   `json:"symbol"`
	Name             string   `json:"name"`
	Sector           string   `json:"sector"`
	Industry         string   `json:"industry"`
	Description      string   `json:"description"`
	Employees        int64    `json:"employees"`
	Website          string   `json:"website"`
	Currency         string   `json:"currency"`
	PE               *float64 `json:"pe,omitempty"`
	EPS              *float64 `json:"eps,omitempty"`
	DividendYield    *float64 `json:"dividendYield,omitempty"`
	FiftyTwoWeekHigh *float64 `json:"52wHigh,omitempty"`
	FiftyTwoWeekLow  *float64 `json:"52wLow,omitempty"`
	Beta             *float64 `json:"beta,omitempty"`
}

type SymbolSearchResult struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Type     string `json:"type"`
}

type WatchlistItem struct {
	Symbol  string    `json:"symbol"`
	Name    string    `json:"name"`
	AddedAt time.Time `json:"addedAt"`
}

type AIInsight struct {
	Symbol         string    `json:"symbol"`
	Summary        string    `json:"summary"`
	Sentiment      string    `json:"sentiment"`
	KeyPoints      []string  `json:"keyPoints"`
	Risks          []string  `json:"risks"`
	Opportunities  []string  `json:"opportunities"`
	Recommendation string    `json:"recommendation"`
	GeneratedAt    time.Time `json:"generatedAt"`
	Provider       string    `json:"provider"`
}

const (
	WSMessageTypeSubscribe   = "subscribe"
	WSMessageTypeUnsubscribe = "unsubscribe"
	WSMessageTypePriceUpdate = "priceUpdate"

	NewsCategoryCompany      = "company"
	NewsCategorySector       = "sector"
	NewsCategoryMarket       = "market"
	NewsCategoryGeopolitical = "geopolitical"
)

type WSClientMessage struct {
	Type   string `json:"type"`
	Symbol string `json:"symbol"`
}

type WSPriceUpdateMessage struct {
	Type      string     `json:"type"`
	Symbol    string     `json:"symbol"`
	Quote     StockQuote `json:"quote"`
	Timestamp time.Time  `json:"timestamp"`
}

type RecommendationTrend struct {
	Period     string `json:"period"`
	StrongBuy  int    `json:"strongBuy"`
	Buy        int    `json:"buy"`
	Hold       int    `json:"hold"`
	Sell       int    `json:"sell"`
	StrongSell int    `json:"strongSell"`
}

type RecommendationData struct {
	Symbol             string                `json:"symbol"`
	RecommendationKey  string                `json:"recommendationKey"`
	RecommendationMean float64               `json:"recommendationMean"`
	NumberOfAnalysts   int                   `json:"numberOfAnalysts"`
	TargetMeanPrice    *float64              `json:"targetMeanPrice,omitempty"`
	TargetHighPrice    *float64              `json:"targetHighPrice,omitempty"`
	TargetLowPrice     *float64              `json:"targetLowPrice,omitempty"`
	CurrentPrice       float64               `json:"currentPrice"`
	Currency           string                `json:"currency"`
	Trend              []RecommendationTrend `json:"trend"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
