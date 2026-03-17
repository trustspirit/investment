package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/shinyoung/investment/internal/model"
)

const (
	googleNewsRSSURL = "https://news.google.com/rss/search"
)

type NewsService struct {
	yahoo  *YahooService
	ai     AIProvider
	client *http.Client
}

func NewNewsService(yahoo *YahooService, ai AIProvider) *NewsService {
	return &NewsService{
		yahoo:  yahoo,
		ai:     ai,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// rssResponse matches RSS 2.0 XML — Google News returns this format.
type rssResponse struct {
	XMLName xml.Name `xml:"rss"`
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title   string `xml:"title"`
	Link    string `xml:"link"`
	PubDate string `xml:"pubDate"`
	Source  string `xml:"source"`
}

func (ns *NewsService) GetAllNews(ctx context.Context, symbol string, sector string, industry string) ([]model.NewsArticle, error) {
	type result struct {
		articles []model.NewsArticle
		err      error
		label    string
	}

	ch := make(chan result, 4)

	go func() {
		articles, err := ns.yahoo.GetNews(ctx, symbol)
		ch <- result{articles: articles, err: err, label: "company"}
	}()

	go func() {
		query := buildSectorQuery(sector, industry)
		if query == "" {
			ch <- result{label: "sector"}
			return
		}
		articles, err := ns.fetchGoogleNews(ctx, query, "en", "US", model.NewsCategorySector, 8)
		ch <- result{articles: articles, err: err, label: "sector"}
	}()

	go func() {
		query := buildMarketQuery(symbol)
		articles, err := ns.fetchGoogleNews(ctx, query, "en", "US", model.NewsCategoryMarket, 5)
		ch <- result{articles: articles, err: err, label: "market"}
	}()

	go func() {
		query := buildKoreanQuery(symbol, sector)
		articles, err := ns.fetchGoogleNews(ctx, query, "ko", "KR", model.NewsCategoryGeopolitical, 5)
		ch <- result{articles: articles, err: err, label: "korean"}
	}()

	var allArticles []model.NewsArticle
	for i := 0; i < 4; i++ {
		r := <-ch
		if r.err != nil {
			slog.Warn("failed to fetch news", "source", r.label, "symbol", symbol, "error", r.err)
			continue
		}
		allArticles = append(allArticles, r.articles...)
	}

	allArticles = deduplicateNews(allArticles)

	sort.Slice(allArticles, func(i, j int) bool {
		return allArticles[i].PublishedAt.After(allArticles[j].PublishedAt)
	})

	if len(allArticles) > 30 {
		allArticles = allArticles[:30]
	}

	allArticles = ns.applySentiment(ctx, allArticles)

	slog.Info("fetched aggregated news", "symbol", symbol, "total", len(allArticles))
	return allArticles, nil
}

func (ns *NewsService) GetBroadNews(ctx context.Context, symbol string, sector string, industry string) []model.NewsArticle {
	type result struct {
		articles []model.NewsArticle
		err      error
		label    string
	}

	ch := make(chan result, 3)

	go func() {
		query := buildSectorQuery(sector, industry)
		if query == "" {
			ch <- result{label: "sector"}
			return
		}
		articles, err := ns.fetchGoogleNews(ctx, query, "en", "US", model.NewsCategorySector, 5)
		ch <- result{articles: articles, err: err, label: "sector"}
	}()

	go func() {
		query := buildMarketQuery(symbol)
		articles, err := ns.fetchGoogleNews(ctx, query, "en", "US", model.NewsCategoryMarket, 5)
		ch <- result{articles: articles, err: err, label: "market"}
	}()

	go func() {
		query := buildKoreanQuery(symbol, sector)
		articles, err := ns.fetchGoogleNews(ctx, query, "ko", "KR", model.NewsCategoryGeopolitical, 3)
		ch <- result{articles: articles, err: err, label: "korean"}
	}()

	var articles []model.NewsArticle
	for i := 0; i < 3; i++ {
		r := <-ch
		if r.err != nil {
			slog.Warn("failed to fetch broad news", "source", r.label, "symbol", symbol, "error", r.err)
			continue
		}
		articles = append(articles, r.articles...)
	}

	return articles
}

func (ns *NewsService) fetchGoogleNews(ctx context.Context, query string, lang string, country string, category string, maxResults int) ([]model.NewsArticle, error) {
	u := fmt.Sprintf("%s?q=%s&hl=%s&gl=%s&ceid=%s:%s",
		googleNewsRSSURL,
		url.QueryEscape(query),
		lang,
		country,
		country,
		lang,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create google news request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)")

	resp, err := ns.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch google news RSS: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google news response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google news returned status %d", resp.StatusCode)
	}

	var rss rssResponse
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("parse google news RSS: %w", err)
	}

	articles := make([]model.NewsArticle, 0, maxResults)
	for _, item := range rss.Channel.Items {
		if len(articles) >= maxResults {
			break
		}

		publishedAt := parseRSSDate(item.PubDate)
		source := item.Source
		if source == "" {
			source = "Google News"
		}

		articles = append(articles, model.NewsArticle{
			Title:          cleanHTMLTitle(item.Title),
			Link:           item.Link,
			Source:         source,
			PublishedAt:    publishedAt,
			Thumbnail:      "",
			RelatedSymbols: nil,
			Category:       category,
		})
	}

	return articles, nil
}

func buildSectorQuery(sector string, industry string) string {
	if sector == "" && industry == "" {
		return ""
	}
	parts := make([]string, 0, 2)
	if industry != "" {
		parts = append(parts, industry)
	}
	if sector != "" && sector != industry {
		parts = append(parts, sector)
	}
	return strings.Join(parts, " ") + " stock market"
}

func buildMarketQuery(symbol string) string {
	base := "stock market economy"
	if isKoreanStock(symbol) {
		base = "Korea stock market KOSPI economy"
	}
	return base
}

func buildKoreanQuery(symbol string, sector string) string {
	if isKoreanStock(symbol) {
		return symbol + " 주식 시장 뉴스"
	}
	cleanSymbol := strings.TrimSuffix(strings.TrimSuffix(symbol, ".KS"), ".KQ")
	query := cleanSymbol + " 주식"
	if sector != "" {
		query += " " + sector
	}
	return query
}

func isKoreanStock(symbol string) bool {
	return strings.HasSuffix(symbol, ".KS") || strings.HasSuffix(symbol, ".KQ")
}

func parseRSSDate(dateStr string) time.Time {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 02 Jan 2006 15:04:05 -0700",
		"Mon, 02 Jan 2006 15:04:05 GMT",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			return t.UTC()
		}
	}
	return time.Now().UTC()
}

func cleanHTMLTitle(title string) string {
	return strings.TrimSpace(title)
}

func analyzeSentiment(title string) string {
	lower := strings.ToLower(title)

	positiveKeywords := []string{
		// English
		"surge", "surges", "soar", "soars", "rally", "rallies", "gain", "gains",
		"rise", "rises", "jump", "jumps", "boom", "booms", "record high",
		"beat", "beats", "outperform", "upgrade", "upgraded", "bullish",
		"profit", "growth", "strong", "recovery", "recover", "rebound",
		"positive", "optimistic", "breakthrough", "innovation", "deal",
		"partnership", "expansion", "dividend", "buyback", "approval",
		"upbeat", "exceeds", "exceeded", "all-time high", "breakout",
		// Korean
		"상승", "급등", "호재", "성장", "흑자", "호실적", "최고가",
		"강세", "반등", "회복", "돌파", "수혜", "긍정", "매수",
		"기대", "확대", "증가", "개선", "호조", "사상최고",
	}

	negativeKeywords := []string{
		// English
		"crash", "crashes", "plunge", "plunges", "drop", "drops", "fall", "falls",
		"decline", "declines", "tumble", "tumbles", "slump", "slumps", "sell-off",
		"selloff", "loss", "losses", "miss", "misses", "downgrade", "downgraded",
		"bearish", "recession", "fear", "fears", "risk", "risks", "warning",
		"warn", "warns", "concern", "crisis", "default", "layoff", "layoffs",
		"cut", "cuts", "shutdown", "bankruptcy", "fraud", "investigation",
		"sanction", "sanctions", "tariff", "tariffs", "war", "conflict",
		"inflation", "rate hike", "deficit", "debt",
		// Korean
		"하락", "급락", "폭락", "악재", "적자", "부진", "손실",
		"약세", "위기", "우려", "경고", "리스크", "불안", "매도",
		"감소", "축소", "하향", "침체", "파산", "제재", "관세",
		"전쟁", "갈등", "인플레이션", "금리인상",
	}

	positiveScore := 0
	negativeScore := 0

	for _, kw := range positiveKeywords {
		if strings.Contains(lower, kw) {
			positiveScore++
		}
	}
	for _, kw := range negativeKeywords {
		if strings.Contains(lower, kw) {
			negativeScore++
		}
	}

	if positiveScore > negativeScore {
		return model.NewsSentimentPositive
	}
	if negativeScore > positiveScore {
		return model.NewsSentimentNegative
	}
	return model.NewsSentimentNeutral
}

func (ns *NewsService) applySentiment(ctx context.Context, articles []model.NewsArticle) []model.NewsArticle {
	titles := make([]string, len(articles))
	for i, a := range articles {
		titles[i] = a.Title
	}

	sentiments, err := ns.ai.AnalyzeSentiment(ctx, titles)
	if err != nil {
		slog.Warn("AI sentiment analysis failed, falling back to keywords", "error", err)
		for i := range articles {
			articles[i].Sentiment = analyzeSentiment(articles[i].Title)
		}
		return articles
	}

	slog.Info("AI sentiment analysis completed", "count", len(sentiments))
	for i := range articles {
		articles[i].Sentiment = sentiments[i]
	}
	return articles
}

func deduplicateNews(articles []model.NewsArticle) []model.NewsArticle {
	seen := make(map[string]bool, len(articles))
	result := make([]model.NewsArticle, 0, len(articles))
	for _, a := range articles {
		key := strings.ToLower(strings.TrimSpace(a.Title))
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, a)
	}
	return result
}
