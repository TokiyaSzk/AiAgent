package timer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aiagent/pkg/base"
	"github.com/aiagent/pkg/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/tmc/langchaingo/llms"
)

func TimerSummaryMemory(rdb *redis.Client, db *pgxpool.Pool, user string) {
	ctx := context.Background()
	llm, err := base.CreateLLMClient()
	if err != nil {
		fmt.Printf("Error creating LLM: %s", err)
	}
	messages := []llms.MessageContent{}
	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, "总结对话历史，返回一段作为记忆体的内容"))

	dailyChatList, err := sql.GetDailyChatMessage(ctx, rdb, user)

	if err != nil {
		fmt.Printf("Error getting daily chat message: %s", err)
	}

	var doc string
	doc += "以下是对话历史："
	for _, message := range dailyChatList {
		var msg string
		err = json.Unmarshal([]byte(message), &msg)
		if err != nil {
			fmt.Printf("Error unmarshalling message: %s", err)
		}
		doc += msg
	}

	messages = append(messages, llms.TextParts(llms.ChatMessageTypeSystem, doc))

	result, err := llm.GenerateContent(ctx, messages)

	if err != nil {
		fmt.Printf("Error generating content: %s", err)
	}

	fmt.Printf("Result: %s\n", result.Choices[0].Content)
}
