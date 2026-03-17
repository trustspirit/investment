package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/openai/openai-go"
	openaiopt "github.com/openai/openai-go/option"

	"github.com/shinyoung/investment/internal/model"
)

type AIProvider interface {
	GenerateInsight(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData) (model.AIInsight, error)
	GenerateStrategy(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData, history []model.HistoricalDataPoint) (model.AITradeStrategy, error)
	ProviderName() string
}

type insightResponse struct {
	Summary        string   `json:"summary"`
	Sentiment      string   `json:"sentiment"`
	KeyPoints      []string `json:"keyPoints"`
	Risks          []string `json:"risks"`
	Opportunities  []string `json:"opportunities"`
	Recommendation string   `json:"recommendation"`
}

func buildPrompt(symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Analyze the investment potential of %s (%s).\n\n", symbol, info.Name))
	sb.WriteString("Company Info:\n")
	sb.WriteString(fmt.Sprintf("- Sector: %s, Industry: %s\n", info.Sector, info.Industry))
	sb.WriteString(fmt.Sprintf("- Employees: %d\n", info.Employees))
	if info.PE != nil {
		sb.WriteString(fmt.Sprintf("- P/E Ratio: %.2f\n", *info.PE))
	}
	if info.EPS != nil {
		sb.WriteString(fmt.Sprintf("- EPS: %.2f\n", *info.EPS))
	}
	if info.Beta != nil {
		sb.WriteString(fmt.Sprintf("- Beta: %.2f\n", *info.Beta))
	}
	if info.DividendYield != nil {
		sb.WriteString(fmt.Sprintf("- Dividend Yield: %.2f%%\n", *info.DividendYield*100))
	}

	currencySymbol := "$"
	if quote.Currency == "KRW" {
		currencySymbol = "₩"
	}
	sb.WriteString(fmt.Sprintf("\nCurrent Price: %s%.2f (Change: %.2f, %.2f%%)\n", currencySymbol, quote.Price, quote.Change, quote.ChangePercent))
	sb.WriteString(fmt.Sprintf("Volume: %d, Market Cap: %d\n", quote.Volume, quote.MarketCap))

	if len(news) > 0 {
		sb.WriteString("\nCompany-Specific News:\n")
		for i, n := range news {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s (Source: %s, %s)\n", n.Title, n.Source, n.PublishedAt.Format(time.RFC3339)))
		}
	}

	if len(broadNews) > 0 {
		sb.WriteString("\nBroader Market/Sector/Geopolitical Context:\n")
		for i, n := range broadNews {
			if i >= 8 {
				break
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s (Source: %s, %s)\n", n.Category, n.Title, n.Source, n.PublishedAt.Format(time.RFC3339)))
		}
	}

	if recommendation != nil && recommendation.RecommendationKey != "" {
		sb.WriteString("\nAnalyst Recommendations:\n")
		sb.WriteString(fmt.Sprintf("- Overall Rating: %s (Mean Score: %.2f / 5.0, 1=Strong Buy, 5=Strong Sell)\n", recommendation.RecommendationKey, recommendation.RecommendationMean))
		sb.WriteString(fmt.Sprintf("- Number of Analysts: %d\n", recommendation.NumberOfAnalysts))
		if recommendation.TargetMeanPrice != nil {
			sb.WriteString(fmt.Sprintf("- Target Mean Price: %.2f\n", *recommendation.TargetMeanPrice))
		}
		if recommendation.TargetHighPrice != nil {
			sb.WriteString(fmt.Sprintf("- Target High Price: %.2f\n", *recommendation.TargetHighPrice))
		}
		if recommendation.TargetLowPrice != nil {
			sb.WriteString(fmt.Sprintf("- Target Low Price: %.2f\n", *recommendation.TargetLowPrice))
		}
		if len(recommendation.Trend) > 0 {
			t := recommendation.Trend[0]
			sb.WriteString(fmt.Sprintf("- Current Period Breakdown: Strong Buy=%d, Buy=%d, Hold=%d, Sell=%d, Strong Sell=%d\n",
				t.StrongBuy, t.Buy, t.Hold, t.Sell, t.StrongSell))
		}
	}

	sb.WriteString(`
Consider the broader market context — including geopolitical events, macroeconomic trends,
trade policies, Fed decisions, sector-wide developments, and related company performance —
when forming your analysis. These factors may significantly impact the stock's outlook.

Provide your analysis as JSON with these exact fields.
All values MUST be written in Korean (한국어).
{
  "summary": "투자 논점에 대한 2-3문장 요약 (회사 뉴스뿐 아니라 시장/지정학적 맥락도 포함)",
  "sentiment": "bullish" or "bearish" or "neutral",
  "keyPoints": ["핵심 포인트1", "핵심 포인트2", "핵심 포인트3"],
  "risks": ["리스크1 (거시경제/지정학적 리스크 포함)", "리스크2", "리스크3"],
  "opportunities": ["기회1", "기회2", "기회3"],
  "recommendation": "간결한 투자 제안"
}

Respond ONLY with valid JSON, no markdown formatting or explanation.`)

	return sb.String()
}

func parseInsightJSON(raw string) (insightResponse, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result insightResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return insightResponse{}, fmt.Errorf("parse AI insight JSON: %w", err)
	}
	return result, nil
}

type strategyResponse struct {
	Signal             string   `json:"signal"`
	Confidence         int      `json:"confidence"`
	EntryPriceLow      float64  `json:"entryPriceLow"`
	EntryPriceHigh     float64  `json:"entryPriceHigh"`
	EntryReason        string   `json:"entryReason"`
	StopLossLow        float64  `json:"stopLossLow"`
	StopLossHigh       float64  `json:"stopLossHigh"`
	StopLossReason     string   `json:"stopLossReason"`
	TakeProfitLow      float64  `json:"takeProfitLow"`
	TakeProfitHigh     float64  `json:"takeProfitHigh"`
	TakeProfitReason   string   `json:"takeProfitReason"`
	BuyRecommendation  string   `json:"buyRecommendation"`
	BuyTimeframe       string   `json:"buyTimeframe"`
	BuyConditions      []string `json:"buyConditions"`
	SellRecommendation string   `json:"sellRecommendation"`
	SellTimeframe      string   `json:"sellTimeframe"`
	SellConditions     []string `json:"sellConditions"`
	RiskReward         string   `json:"riskReward"`
	AnalysisBasis      []string `json:"analysisBasis"`
	MarketCondition    string   `json:"marketCondition"`
	ShortTermView      string   `json:"shortTermView"`
	MidTermView        string   `json:"midTermView"`
}

func buildStrategyPrompt(symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData, history []model.HistoricalDataPoint) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("현재 시각: %s\n\n", time.Now().Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("종목: %s (%s)\n", symbol, info.Name))
	sb.WriteString(fmt.Sprintf("섹터: %s, 산업: %s\n\n", info.Sector, info.Industry))

	currencySymbol := "$"
	if quote.Currency == "KRW" {
		currencySymbol = "₩"
	}

	sb.WriteString(fmt.Sprintf("현재가: %s%.2f (변동: %.2f, %.2f%%)\n", currencySymbol, quote.Price, quote.Change, quote.ChangePercent))
	sb.WriteString(fmt.Sprintf("거래량: %d, 시가총액: %d\n", quote.Volume, quote.MarketCap))

	if info.FiftyTwoWeekHigh != nil && info.FiftyTwoWeekLow != nil {
		sb.WriteString(fmt.Sprintf("52주 최고: %s%.2f, 52주 최저: %s%.2f\n", currencySymbol, *info.FiftyTwoWeekHigh, currencySymbol, *info.FiftyTwoWeekLow))
	}
	if info.PE != nil {
		sb.WriteString(fmt.Sprintf("PER: %.2f\n", *info.PE))
	}
	if info.EPS != nil {
		sb.WriteString(fmt.Sprintf("EPS: %.2f\n", *info.EPS))
	}
	if info.Beta != nil {
		sb.WriteString(fmt.Sprintf("베타: %.2f\n", *info.Beta))
	}

	if len(history) > 0 {
		sb.WriteString("\n최근 가격 추이 (최근 10개 데이터):\n")
		start := 0
		if len(history) > 10 {
			start = len(history) - 10
		}
		for _, dp := range history[start:] {
			sb.WriteString(fmt.Sprintf("  %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%d\n",
				dp.Timestamp.Format("2006-01-02 15:04"), dp.Open, dp.High, dp.Low, dp.Close, dp.Volume))
		}
	}

	if recommendation != nil && recommendation.RecommendationKey != "" {
		sb.WriteString(fmt.Sprintf("\n애널리스트 컨센서스: %s (평균 점수: %.2f/5.0, 애널리스트 수: %d)\n",
			recommendation.RecommendationKey, recommendation.RecommendationMean, recommendation.NumberOfAnalysts))
		if recommendation.TargetMeanPrice != nil {
			sb.WriteString(fmt.Sprintf("목표가 평균: %s%.2f\n", currencySymbol, *recommendation.TargetMeanPrice))
		}
		if recommendation.TargetHighPrice != nil {
			sb.WriteString(fmt.Sprintf("목표가 최고: %s%.2f\n", currencySymbol, *recommendation.TargetHighPrice))
		}
		if recommendation.TargetLowPrice != nil {
			sb.WriteString(fmt.Sprintf("목표가 최저: %s%.2f\n", currencySymbol, *recommendation.TargetLowPrice))
		}
	}

	if len(news) > 0 {
		sb.WriteString("\n주요 뉴스:\n")
		for i, n := range news {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- %s (%s)\n", n.Title, n.Source))
		}
	}

	if len(broadNews) > 0 {
		sb.WriteString("\n시장/지정학적 뉴스:\n")
		for i, n := range broadNews {
			if i >= 5 {
				break
			}
			sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", n.Category, n.Title, n.Source))
		}
	}

	sb.WriteString(`

위 데이터를 기반으로 종합적인 투자 전략을 분석해 주세요.

다음 기준을 모두 고려하세요:
1. 기술적 분석: 최근 가격 추이, 거래량 패턴, 지지선/저항선
2. 펀더멘털 분석: PER, EPS, 시가총액, 배당수익률
3. 시장 심리: 뉴스 동향, 애널리스트 의견, 시장 전체 분위기
4. 거시경제: 금리, 환율, 지정학적 리스크
5. 리스크/보상 비율

JSON으로 응답해 주세요. 모든 값은 한국어로 작성하세요.
{
  "signal": "적극매수" | "매수" | "중립" | "매도" | "적극매도",
  "confidence": 0-100 사이 정수 (분석 확신도),
  "entryPriceLow": 진입 가격대 하한 (숫자),
  "entryPriceHigh": 진입 가격대 상한 (숫자),
  "entryReason": "해당 가격대를 추천하는 근거",
  "stopLossLow": 손절 가격대 하한 (숫자),
  "stopLossHigh": 손절 가격대 상한 (숫자),
  "stopLossReason": "손절 기준 근거",
  "takeProfitLow": 익절 가격대 하한 (숫자),
  "takeProfitHigh": 익절 가격대 상한 (숫자),
  "takeProfitReason": "익절 목표 근거",
  "buyRecommendation": "매수 타이밍에 대한 구체적 제안",
  "buyTimeframe": "단기(1-2주)" | "중기(1-3개월)" | "장기(6개월+)",
  "buyConditions": ["매수 조건1", "매수 조건2", "매수 조건3"],
  "sellRecommendation": "매도 타이밍에 대한 구체적 제안",
  "sellTimeframe": "단기(1-2주)" | "중기(1-3개월)" | "장기(6개월+)",
  "sellConditions": ["매도 조건1", "매도 조건2", "매도 조건3"],
  "riskReward": "리스크 대비 보상 비율 (예: 1:2.5)",
  "analysisBasis": ["판단 근거1", "판단 근거2", "판단 근거3", "판단 근거4"],
  "marketCondition": "현재 시장 상황에 대한 종합 평가",
  "shortTermView": "단기(1-2주) 전망 및 전략",
  "midTermView": "중기(1-3개월) 전망 및 전략"
}

Respond ONLY with valid JSON, no markdown formatting or explanation.`)

	return sb.String()
}

func parseStrategyJSON(raw string) (strategyResponse, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var result strategyResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return strategyResponse{}, fmt.Errorf("parse AI strategy JSON: %w", err)
	}
	return result, nil
}

func strategyFromResponse(symbol string, parsed strategyResponse, quote model.StockQuote, provider string) model.AITradeStrategy {
	return model.AITradeStrategy{
		Symbol:          symbol,
		AnalysisTime:    time.Now().UTC(),
		CurrentPrice:    quote.Price,
		Currency:        quote.Currency,
		Signal:          parsed.Signal,
		Confidence:      parsed.Confidence,
		EntryPrice:      model.PriceRange{Low: parsed.EntryPriceLow, High: parsed.EntryPriceHigh, Reason: parsed.EntryReason},
		StopLoss:        model.PriceRange{Low: parsed.StopLossLow, High: parsed.StopLossHigh, Reason: parsed.StopLossReason},
		TakeProfit:      model.PriceRange{Low: parsed.TakeProfitLow, High: parsed.TakeProfitHigh, Reason: parsed.TakeProfitReason},
		BuyTiming:       model.TimingAnalysis{Recommendation: parsed.BuyRecommendation, Timeframe: parsed.BuyTimeframe, Conditions: parsed.BuyConditions},
		SellTiming:      model.TimingAnalysis{Recommendation: parsed.SellRecommendation, Timeframe: parsed.SellTimeframe, Conditions: parsed.SellConditions},
		RiskReward:      parsed.RiskReward,
		AnalysisBasis:   parsed.AnalysisBasis,
		MarketCondition: parsed.MarketCondition,
		ShortTermView:   parsed.ShortTermView,
		MidTermView:     parsed.MidTermView,
		Disclaimer:      "본 분석은 AI가 생성한 참고 자료이며, 투자 결정의 최종 책임은 투자자 본인에게 있습니다.",
		Provider:        provider,
	}
}

// AnthropicProvider

type AnthropicProvider struct {
	client anthropic.Client
	model  string
}

func NewAnthropicProvider(apiKey string, aiModel string) *AnthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	if aiModel == "" {
		aiModel = "claude-sonnet-4-20250514"
	}
	return &AnthropicProvider{client: client, model: aiModel}
}

func (p *AnthropicProvider) ProviderName() string { return "anthropic" }

func (p *AnthropicProvider) GenerateInsight(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData) (model.AIInsight, error) {
	prompt := buildPrompt(symbol, info, news, broadNews, quote, recommendation)

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 2048,
		Model:     p.model,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return model.AIInsight{}, fmt.Errorf("anthropic API call for %s: %w", symbol, err)
	}

	if len(message.Content) == 0 {
		return model.AIInsight{}, fmt.Errorf("empty response from anthropic for %s", symbol)
	}

	rawText := ""
	for _, block := range message.Content {
		if block.Type == "text" {
			rawText = block.Text
			break
		}
	}

	parsed, err := parseInsightJSON(rawText)
	if err != nil {
		slog.Warn("failed to parse anthropic response", "symbol", symbol, "raw", rawText[:min(len(rawText), 200)])
		return model.AIInsight{}, fmt.Errorf("parse anthropic response for %s: %w", symbol, err)
	}

	return model.AIInsight{
		Symbol:         symbol,
		Summary:        parsed.Summary,
		Sentiment:      parsed.Sentiment,
		KeyPoints:      parsed.KeyPoints,
		Risks:          parsed.Risks,
		Opportunities:  parsed.Opportunities,
		Recommendation: parsed.Recommendation,
		GeneratedAt:    time.Now().UTC(),
		Provider:       p.ProviderName(),
	}, nil
}

func (p *AnthropicProvider) GenerateStrategy(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData, history []model.HistoricalDataPoint) (model.AITradeStrategy, error) {
	prompt := buildStrategyPrompt(symbol, info, news, broadNews, quote, recommendation, history)

	message, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 4096,
		Model:     p.model,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return model.AITradeStrategy{}, fmt.Errorf("anthropic strategy call for %s: %w", symbol, err)
	}

	if len(message.Content) == 0 {
		return model.AITradeStrategy{}, fmt.Errorf("empty strategy response from anthropic for %s", symbol)
	}

	rawText := ""
	for _, block := range message.Content {
		if block.Type == "text" {
			rawText = block.Text
			break
		}
	}

	parsed, err := parseStrategyJSON(rawText)
	if err != nil {
		slog.Warn("failed to parse anthropic strategy response", "symbol", symbol, "raw", rawText[:min(len(rawText), 200)])
		return model.AITradeStrategy{}, fmt.Errorf("parse anthropic strategy for %s: %w", symbol, err)
	}

	return strategyFromResponse(symbol, parsed, quote, p.ProviderName()), nil
}

// OpenAIProvider

type OpenAIProvider struct {
	client openai.Client
	model  string
}

func NewOpenAIProvider(apiKey string, aiModel string) *OpenAIProvider {
	client := openai.NewClient(openaiopt.WithAPIKey(apiKey))
	if aiModel == "" {
		aiModel = "gpt-4o"
	}
	return &OpenAIProvider{client: client, model: aiModel}
}

func (p *OpenAIProvider) ProviderName() string { return "openai" }

func (p *OpenAIProvider) GenerateInsight(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData) (model.AIInsight, error) {
	prompt := buildPrompt(symbol, info, news, broadNews, quote, recommendation)

	completion, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return model.AIInsight{}, fmt.Errorf("openai API call for %s: %w", symbol, err)
	}

	if len(completion.Choices) == 0 {
		return model.AIInsight{}, fmt.Errorf("empty response from openai for %s", symbol)
	}

	rawText := completion.Choices[0].Message.Content
	parsed, err := parseInsightJSON(rawText)
	if err != nil {
		slog.Warn("failed to parse openai response", "symbol", symbol, "raw", rawText[:min(len(rawText), 200)])
		return model.AIInsight{}, fmt.Errorf("parse openai response for %s: %w", symbol, err)
	}

	return model.AIInsight{
		Symbol:         symbol,
		Summary:        parsed.Summary,
		Sentiment:      parsed.Sentiment,
		KeyPoints:      parsed.KeyPoints,
		Risks:          parsed.Risks,
		Opportunities:  parsed.Opportunities,
		Recommendation: parsed.Recommendation,
		GeneratedAt:    time.Now().UTC(),
		Provider:       p.ProviderName(),
	}, nil
}

func (p *OpenAIProvider) GenerateStrategy(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote, recommendation *model.RecommendationData, history []model.HistoricalDataPoint) (model.AITradeStrategy, error) {
	prompt := buildStrategyPrompt(symbol, info, news, broadNews, quote, recommendation, history)

	completion, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: p.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	})
	if err != nil {
		return model.AITradeStrategy{}, fmt.Errorf("openai strategy call for %s: %w", symbol, err)
	}

	if len(completion.Choices) == 0 {
		return model.AITradeStrategy{}, fmt.Errorf("empty strategy response from openai for %s", symbol)
	}

	rawText := completion.Choices[0].Message.Content
	parsed, err := parseStrategyJSON(rawText)
	if err != nil {
		slog.Warn("failed to parse openai strategy response", "symbol", symbol, "raw", rawText[:min(len(rawText), 200)])
		return model.AITradeStrategy{}, fmt.Errorf("parse openai strategy for %s: %w", symbol, err)
	}

	return strategyFromResponse(symbol, parsed, quote, p.ProviderName()), nil
}

// NoopProvider is used when no AI API key is configured.

type NoopProvider struct{}

func (p *NoopProvider) ProviderName() string { return "none" }

func (p *NoopProvider) GenerateInsight(_ context.Context, symbol string, _ model.CompanyInfo, _ []model.NewsArticle, _ []model.NewsArticle, _ model.StockQuote, _ *model.RecommendationData) (model.AIInsight, error) {
	return model.AIInsight{}, fmt.Errorf("AI provider not configured — set ANTHROPIC_API_KEY or OPENAI_API_KEY in .env to enable AI insights for %s", symbol)
}

func (p *NoopProvider) GenerateStrategy(_ context.Context, symbol string, _ model.CompanyInfo, _ []model.NewsArticle, _ []model.NewsArticle, _ model.StockQuote, _ *model.RecommendationData, _ []model.HistoricalDataPoint) (model.AITradeStrategy, error) {
	return model.AITradeStrategy{}, fmt.Errorf("AI provider not configured — set ANTHROPIC_API_KEY or OPENAI_API_KEY in .env to enable AI strategy for %s", symbol)
}

// Factory

func NewAIProvider(provider, apiKey, aiModel string) (AIProvider, error) {
	switch strings.ToLower(provider) {
	case "anthropic":
		return NewAnthropicProvider(apiKey, aiModel), nil
	case "openai":
		return NewOpenAIProvider(apiKey, aiModel), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider: %s", provider)
	}
}
