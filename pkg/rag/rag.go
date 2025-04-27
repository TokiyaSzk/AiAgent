package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/aiagent/pkg/base"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

type MemoryItem struct {
	Content  string
	Distance float32
}

func InitEmbedder() (*embeddings.EmbedderImpl, error) {
	config, err := base.GetEnv()

	if err != nil {
		return nil, err
	}

	llm, err := openai.New(
		openai.WithToken(config.ApiKey),
		openai.WithBaseURL(config.BaseUrl),
		openai.WithEmbeddingModel("text-embedding-v1"),
	)

	if err != nil {
		return nil, err
	}
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		return nil, err
	}
	return embedder, nil
}

func EmbedText(ctx context.Context, text string, embedder *embeddings.EmbedderImpl) ([]float64, error) {
	embs, err := embedder.EmbedDocuments(ctx, []string{text})

	if err != nil {
		return nil, err
	}
	return Float32To64(embs[0]), nil
}

func InsertDocument(ctx context.Context, db *pgxpool.Pool, content string, embedder *embeddings.EmbedderImpl) error {
	vec, err := EmbedText(ctx, content, embedder)
	if err != nil {
		return fmt.Errorf("error embedding document: %w", err)
	}

	vectorStr := Float64ArrayToPGVector(vec)

	query := `INSERT INTO documents (content, embedding) VALUES ($1, $2)`
	_, err = db.Exec(ctx, query, content, vectorStr)

	return err
}

func InsertMemory(ctx context.Context, db *pgxpool.Pool, content string, embedder *embeddings.EmbedderImpl) error {
	vec, err := EmbedText(ctx, content, embedder)
	if err != nil {
		return fmt.Errorf("error embedding document: %w", err)
	}

	vectorStr := Float64ArrayToPGVector(vec)

	query := `INSERT INTO memory (content, embedding) VALUES ($1, $2)`
	_, err = db.Exec(ctx, query, content, vectorStr)

	return err
}

func RetrieveRelevantMemory(ctx context.Context, queryVec []float64, topK int, db *pgxpool.Pool) ([]string, error) {
	vector := pgvector.NewVector(Float64To32(queryVec))
	var results []string
	sqlStr := `
    SELECT content, embedding <-> $1 AS distance
    FROM memory
    ORDER BY distance
    LIMIT $2`

	rows, err := db.Query(ctx, sqlStr, vector, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MemoryItem
		if err := rows.Scan(&item.Content, &item.Distance); err != nil {
			return nil, err
		}
		if item.Distance <= 0.5 {
			results = append(results, item.Content)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func RetrieveRelevantDocs(ctx context.Context, queryVec []float64, topK int, db *pgxpool.Pool) ([]string, error) {
	vector := pgvector.NewVector(Float64To32(queryVec))
	var results []string
	sqlStr := `
    SELECT content, embedding <-> $1 AS distance
    FROM documents
    ORDER BY distance
    LIMIT $2`

	rows, err := db.Query(ctx, sqlStr, vector, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MemoryItem
		if err := rows.Scan(&item.Content, &item.Distance); err != nil {
			return nil, err
		}
		if item.Distance <= 0.5 {
			results = append(results, item.Content)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func RunRAG(
	ctx context.Context, question string, topK int,
	embedder *embeddings.EmbedderImpl, db *pgxpool.Pool, llm *openai.LLM) (string, error) {
	queryVec, err := EmbedText(ctx, question, embedder)
	if err != nil {
		return "", err
	}

	relevantDocs, err := RetrieveRelevantDocs(ctx, queryVec, topK, db)
	if err != nil {
		return "", err
	}

	answer, err := RagGenerateAnswer(ctx, relevantDocs, question, llm)
	if err != nil {
		return "", err
	}

	return answer, nil
}

func RagGenerateAnswer(ctx context.Context, docs []string, message string, llm *openai.LLM) (string, error) {
	prompt := fmt.Sprintf("资料如下：\n%s\n\n问题：%s\n请基于上述资料回答：", JoinDocs(docs), message)
	return llm.Call(ctx, prompt)
}

func JoinDocs(docs []string) string {
	joined := ""
	for _, doc := range docs {
		joined += "- " + doc + "\n"
	}
	return joined
}

func ScanDocuments(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
	sqlStr := `SELECT content FROM documents`

	rows, err := db.Query(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []string

	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		docs = append(docs, content)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return docs, nil
}

func ScanMemory(ctx context.Context, db *pgxpool.Pool) ([]string, error) {
	sqlStr := `SELECT content FROM memory`

	rows, err := db.Query(ctx, sqlStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []string

	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		docs = append(docs, content)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return docs, nil
}

// Float transfer
func Float32To64(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, v := range f32 {
		f64[i] = float64(v)
	}
	return f64
}
func Float64To32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, v := range f64 {
		f32[i] = float32(v)
	}
	return f32
}

func Float64ArrayToPGVector(vec []float64) string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, v := range vec {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(fmt.Sprintf("%f", v))
	}
	sb.WriteString("]")
	return sb.String()
}
