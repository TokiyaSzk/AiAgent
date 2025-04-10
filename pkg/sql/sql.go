package sql

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aiagent/pkg/base"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type CharaPrompt struct {
	Name   string
	Prompt string
}

// POSTGRES
func CreatePSQLClient(ctx context.Context) (*pgxpool.Pool, error) {
	config, err := base.GetEnv()
	if err != nil {
		return nil, fmt.Errorf("error loading configuration: %w", err)
	}

	postgresConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	postgresConfig.MaxConns = 10
	postgresConfig.MinConns = 1
	postgresConfig.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, postgresConfig)

	if err != nil {
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	return pool, nil
}

func CreatePSQLDatabase(ctx context.Context, db *pgxpool.Pool) error {

	_, err := db.Exec(ctx, "CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		return fmt.Errorf("error creating vector extension: %w", err)
	}
	return nil
}
func CreatePSQLTable(ctx context.Context, db *pgxpool.Pool) error {
	// 创建表的 SQL 语句
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS chara (
		rid SERIAL PRIMARY KEY,
		role TEXT NOT NULL,
		content TEXT NOT NULL
	);
	`

	// 执行创建表的 SQL 语句
	_, err := db.Exec(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("error creating table: %w", err)
	}
	return nil
}

// REDIS

func CreateRedisClient(ctx context.Context) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         "localhost:6379",
		Password:     "", // no password set
		DB:           1,  // use default DB
		PoolSize:     10,
		MinIdleConns: 5,
		PoolTimeout:  30 * time.Second,
	})
	return rdb, nil
}

func CountCharaPrompt(ctx context.Context, rdb *redis.Client) (int64, error) {
	baseCount := 0
	var cursor uint64
	for {
		// 执行 SCAN 命令
		var keys []string
		var err error
		keys, cursor, err = rdb.Scan(ctx, cursor, "ai:chara:*", 0).Result()
		if err != nil {
			fmt.Printf("Error scanning keys: %v\n", err)
			return 0, err
		}

		// 更新计数器
		baseCount += len(keys)

		// 如果游标为 0，表示遍历完成
		if cursor == 0 {
			break
		}
	}
	// 输出结果
	return int64(baseCount), nil
}

func SaveCharaPrompt(ctx context.Context, rdb *redis.Client, chara string, content string) error {
	baseCount, err := CountCharaPrompt(ctx, rdb)
	if err != nil {
		return fmt.Errorf("error counting chara prompts: %w", err)
	}
	roleID := fmt.Sprint(baseCount + 1)
	key := fmt.Sprintf("ai:chara:%s", roleID)

	res := rdb.HSet(ctx, key, "name", chara, "prompt", content)

	if err := res.Err(); err != nil {
		return err
	}

	rdb.SAdd(ctx, "ai:chara:ids", roleID)
	return nil
}

func RemoveCharaPrompt(ctx context.Context, rdb *redis.Client, roleID string) error {
	key := fmt.Sprintf("ai:chara:%s", roleID)

	// 先检查角色是否存在（非空 Hash 或存在字段）
	exists, err := rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return fmt.Errorf("角色不存在: %s", key)
	}

	// 删除整个角色 Hash
	if err := rdb.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除角色失败: %v", err)
	}

	// 从 ID 集合中移除
	if err := rdb.SRem(ctx, "ai:chara:ids", roleID).Err(); err != nil {
		return fmt.Errorf("从ID集合中移除失败: %v", err)
	}

	fmt.Printf("成功删除角色 %s\n", roleID)
	return nil
}
func CleanInvalidCharaIDs(ctx context.Context, rdb *redis.Client) error {
	ids, err := rdb.SMembers(ctx, "ai:chara:ids").Result()
	if err != nil {
		return err
	}

	for _, id := range ids {
		key := fmt.Sprintf("ai:chara:%s", id)
		exists, _ := rdb.Exists(ctx, key).Result()
		if exists == 0 {
			// 删除失效 ID
			rdb.SRem(ctx, "ai:chara:ids", id)
			fmt.Printf("清理无效角色ID: %s\n", id)
		}
	}
	return nil
}

func GetCharaPrompt(ctx context.Context, rdb *redis.Client, roleID string) (*CharaPrompt, error) {

	result, err := rdb.HGetAll(ctx, roleID).Result()
	if err != nil {
		log.Fatal("Error getting chara prompt from Redis:", err)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no chara found with roleID %s", roleID)
	}
	charaPrompt := &CharaPrompt{
		Name:   result["name"],
		Prompt: result["prompt"],
	}
	return charaPrompt, nil
}

func GetAllCharaIDs(ctx context.Context, rdb *redis.Client) ([]string, error) {
	return rdb.SMembers(ctx, "ai:chara:ids").Result()
}
