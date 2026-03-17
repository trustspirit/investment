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
	GenerateInsight(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote) (model.AIInsight, error)
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

func buildPrompt(symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote) string {
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

func (p *AnthropicProvider) GenerateInsight(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote) (model.AIInsight, error) {
	prompt := buildPrompt(symbol, info, news, broadNews, quote)

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

func (p *OpenAIProvider) GenerateInsight(ctx context.Context, symbol string, info model.CompanyInfo, news []model.NewsArticle, broadNews []model.NewsArticle, quote model.StockQuote) (model.AIInsight, error) {
	prompt := buildPrompt(symbol, info, news, broadNews, quote)

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

// NoopProvider is used when no AI API key is configured.

type NoopProvider struct{}

func (p *NoopProvider) ProviderName() string { return "none" }

func (p *NoopProvider) GenerateInsight(_ context.Context, symbol string, _ model.CompanyInfo, _ []model.NewsArticle, _ []model.NewsArticle, _ model.StockQuote) (model.AIInsight, error) {
	return model.AIInsight{}, fmt.Errorf("AI provider not configured — set ANTHROPIC_API_KEY or OPENAI_API_KEY in .env to enable AI insights for %s", symbol)
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
