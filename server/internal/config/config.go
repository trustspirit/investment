package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ProviderAnthropic = "anthropic"
	ProviderOpenAI    = "openai"
	defaultPort       = "8081"

	defaultAnthropicModel = "claude-sonnet-4-20250514"
	defaultOpenAIModel    = "gpt-4o"
)

type Config struct {
	Port            string
	AnthropicAPIKey string
	OpenAIAPIKey    string
	AIProvider      string
	AIModel         string
}

func Load() (Config, error) {
	dotEnvValues, err := loadDotEnvFiles("server/.env", ".env")
	if err != nil {
		return Config{}, err
	}

	readValue := func(key string) string {
		if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}

		if value, ok := dotEnvValues[key]; ok {
			return strings.TrimSpace(value)
		}

		return ""
	}

	cfg := Config{
		Port:            withDefault(readValue("PORT"), defaultPort),
		AnthropicAPIKey: readValue("ANTHROPIC_API_KEY"),
		OpenAIAPIKey:    readValue("OPENAI_API_KEY"),
		AIProvider:      strings.ToLower(withDefault(readValue("AI_PROVIDER"), ProviderAnthropic)),
		AIModel:         strings.TrimSpace(readValue("AI_MODEL")),
	}

	switch cfg.AIProvider {
	case ProviderAnthropic:
		if cfg.AIModel == "" {
			cfg.AIModel = defaultAnthropicModel
		}
	case ProviderOpenAI:
		if cfg.AIModel == "" {
			cfg.AIModel = defaultOpenAIModel
		}
	default:
		return Config{}, fmt.Errorf("unsupported AI_PROVIDER %q", cfg.AIProvider)
	}

	return cfg, nil
}

func (c Config) Address() string {
	return ":" + c.Port
}

func withDefault(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return strings.TrimSpace(value)
}

func loadDotEnvFiles(paths ...string) (map[string]string, error) {
	values := make(map[string]string)

	for _, path := range paths {
		cleanPath := filepath.Clean(path)
		if _, err := os.Stat(cleanPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return nil, fmt.Errorf("stat dotenv file %s: %w", cleanPath, err)
		}

		parsed, err := parseDotEnv(cleanPath)
		if err != nil {
			return nil, err
		}

		for key, value := range parsed {
			values[key] = value
		}
	}

	return values, nil
}

func parseDotEnv(path string) (map[string]string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("open dotenv file %s: %w", path, err)
	}
	defer file.Close()

	values := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid dotenv format in %s:%d", path, lineNumber)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = strings.Trim(value, `"`)
		value = strings.Trim(value, `'`)

		if key == "" {
			return nil, fmt.Errorf("empty dotenv key in %s:%d", path, lineNumber)
		}

		values[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dotenv file %s: %w", path, err)
	}

	return values, nil
}
