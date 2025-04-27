package mem

import (
	"context"

	"github.com/aiagent/pkg/base"
	"github.com/aiagent/pkg/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func SummaryMemory(ctx context.Context, rdb *redis.Client, db *pgxpool.Pool, sessionID string, user string) (string, error) {

	chatList, err := sql.GetChatMessage(ctx, rdb, sessionID, user)
	if err != nil {
		return "", err
	}
	llm, err := base.CreateLLMClient()

	if err != nil {
		return "", err
	}
	prompt := "以下是对话历史，不要使用任何机械性的词语，对方的名字是" + user +
		"，请以第一人称的视角进行总结成一段作为记忆体的内容："
	for _, msg := range chatList {
		prompt += msg
	}
	result, err := llm.Call(ctx, prompt)
	if err != nil {
		return "", err
	}
	return result, nil
}
