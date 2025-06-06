package base

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms/openai"
)

type Config struct {
	ApiKey      string
	Model       string
	BaseUrl     string
	DatabaseURL string
}

func GetEnv() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, err
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("MODEL_NAME")
	baseUrl := os.Getenv("BASE_URL")
	databaseURL := os.Getenv("DATABASE_URL")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	return Config{
		ApiKey:      apiKey,
		Model:       model,
		BaseUrl:     baseUrl,
		DatabaseURL: databaseURL,
	}, nil
}

func CreateLLMClient() (*openai.LLM, error) {
	// Create a new LLM client using the provided configuration
	config, err := GetEnv()
	if err != nil {
		log.Fatalf("Error loading API key: %s", err)
	}
	llm, err := openai.New(
		openai.WithModel(config.Model),
		openai.WithToken(config.ApiKey),
		openai.WithBaseURL(config.BaseUrl),
	)
	if err != nil {
		log.Fatalf("Error creating LLM: %s", err)
	}
	return llm, nil
}

func GenerateSessionID() string {
	// Generate a unique session ID using the current timestamp
	now := time.Now()
	return now.Format("20060102150405")
}

func isToday(t time.Time) bool {
	now := time.Now()

	// 获取今天 0 点
	year, month, day := now.Date()
	location := now.Location()
	startOfDay := time.Date(year, month, day, 0, 0, 0, 0, location)

	// 获取明天 0 点
	endOfDay := startOfDay.Add(24 * time.Hour)

	// 判断 t 是否在今天范围内
	return t.After(startOfDay) && t.Before(endOfDay)
}
