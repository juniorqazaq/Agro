package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address       string
	KnowledgeBase string
	GeminiAPIKey  string
	GeminiModel   string
	AITimeout     time.Duration
	AIRetries     int
	MockDiseaseID string
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		Address:       env("SERVER_ADDRESS", ":8000"),
		KnowledgeBase: env("KNOWLEDGE_BASE_PATH", "data/disease_database.json"),
		GeminiAPIKey:  os.Getenv("GEMINI_API_KEY"),
		GeminiModel:   env("GEMINI_MODEL", "gemini-2.5-flash"),
		AITimeout:     durationEnv("AI_TIMEOUT", 30*time.Second),
		AIRetries:     intEnv("AI_RETRIES", 10),
		MockDiseaseID: strings.TrimSpace(os.Getenv("AI_MOCK_DISEASE_ID")),
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value < 1 {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	value, err := time.ParseDuration(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key != "" {
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, value)
			}
		}
	}
}
