package test

import (
	"context"
	"testing"
	"time"

	"github.com/aiagent/pkg/sql"
)

func TestGetAllDocument(t *testing.T) {
	ctx := context.Background()

	db, err := sql.CreatePSQLClient(ctx)

	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer db.Close()
	result, err := sql.GetAllDocument(ctx, db)
	if err != nil {
		t.Fatalf("获取所有文档失败: %v", err)
	}
	if len(result) == 0 {
		t.Fatalf("获取的文档数量为0")
	}
	t.Logf("获取到的文档数量: %d", len(result))
	for _, doc := range result {
		t.Logf("文档内容: %s", doc)
	}

}

func TestSaveChatMessage(t *testing.T) {
	ctx := context.Background()

	rdb, err := sql.CreateRedisClient(ctx)

	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	timestamp := time.Now().Unix()
	defer rdb.Close()
	message := sql.Message{
		Role:      "Ai",
		Content:   "I am fine",
		Timestamp: timestamp,
	}

	err = sql.SaveChatMessage(ctx, rdb, message, "12345", "test")
	if err != nil {
		t.Fatalf("保存聊天消息失败: %v", err)
	}

	t.Logf("保存聊天消息成功")
}

func TestGetChatMessage(t *testing.T) {
	ctx := context.Background()

	rdb, err := sql.CreateRedisClient(ctx)

	if err != nil {
		t.Fatalf("创建数据库连接失败: %v", err)
	}
	defer rdb.Close()
	messages, err := sql.GetChatMessage(ctx, rdb, "20250414210843", "tokiya")
	if err != nil {
		t.Fatalf("获取聊天消息失败: %v", err)
	}

	if len(messages) == 0 {
		t.Fatalf("获取的聊天消息数量为0")
	}
	t.Logf("获取到的聊天消息数量: %d", len(messages))
	for _, msg := range messages {
		t.Logf("聊天消息内容: %s", msg)
	}
}
