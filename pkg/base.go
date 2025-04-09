package pkg

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

type Config struct {
	ApiKey  string
	Model   string
	BaseUrl string
}

func GetEnv() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, err
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("MODEL_NAME")
	baseUrl := os.Getenv("BASE_URL")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}
	return Config{
		ApiKey:  apiKey,
		Model:   model,
		BaseUrl: baseUrl,
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

func MakeMessages(role llms.ChatMessageType, message string, messages []llms.MessageContent) ([]llms.MessageContent, error) {
	messages = append(messages, llms.MessageContent{
		Role: role,
		Parts: []llms.ContentPart{
			llms.TextPart(message),
		},
	})
	return messages, nil
}
